package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

// fakeKey is a recognisable fake credential: tests assert it never leaks into
// an error message.
const fakeKey = "AIzaSy-FAKE-TUTORIAL-KEY-0123456789"

// recorded is one request a fake server received.
type recorded struct {
	Method   string
	Path     string
	RawQuery string
	HasKey   bool
	Body     map[string]any
}

// reply is one scripted answer. The last reply repeats.
type reply struct {
	Status int    // 0 = 200
	Body   string // raw body
	SSE    bool   // text/event-stream
	Cut    bool   // announce more bytes than are sent, then drop the connection
	Do     func() // runs before the answer is written
}

// fakeAPI stands in for generateContent, streamGenerateContent and
// interactions: it records every request and answers from a script.
type fakeAPI struct {
	srv     *httptest.Server
	mu      sync.Mutex
	reqs    []recorded
	replies []reply
	dynamic func(n int, r recorded) reply // overrides replies when set
}

func newFake(t *testing.T, replies ...reply) *fakeAPI {
	t.Helper()
	f := &fakeAPI{replies: replies}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		rec := recorded{
			Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery,
			HasKey: r.Header.Get("x-goog-api-key") == fakeKey, Body: body,
		}
		f.mu.Lock()
		f.reqs = append(f.reqs, rec)
		n := len(f.reqs) - 1
		f.mu.Unlock()

		var rp reply
		switch {
		case f.dynamic != nil:
			rp = f.dynamic(n, rec)
		case len(f.replies) == 0:
			rp = reply{Status: 500, Body: `{"error":{"message":"no script"}}`}
		default:
			rp = f.replies[min(n, len(f.replies)-1)]
		}
		if rp.Do != nil {
			rp.Do()
		}
		if rp.SSE {
			w.Header().Set("Content-Type", "text/event-stream")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		if rp.Cut {
			w.Header().Set("Content-Length", fmt.Sprint(len(rp.Body)+4096))
		}
		if rp.Status != 0 {
			w.WriteHeader(rp.Status)
		}
		_, _ = io.WriteString(w, rp.Body)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeAPI) hits() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.reqs)
}

func (f *fakeAPI) req(i int) recorded {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i >= len(f.reqs) {
		return recorded{}
	}
	return f.reqs[i]
}

// geminiClient points a "gemini" (generateContent) client at the fake. The
// base mirrors the real https://.../v1beta/models, so the request path is
// /v1beta/models/<model>:generateContent. Retries are off: one transport
// attempt per call, so hit counts are exact.
func geminiClient(t *testing.T, f *fakeAPI, model string, mods ...func(*llm.Config)) llm.Client {
	t.Helper()
	cfg := llm.Config{
		Provider: "gemini", Model: model, APIKey: fakeKey,
		BaseURL: f.srv.URL + "/v1beta/models", DisableRetries: true,
	}
	for _, m := range mods {
		m(&cfg)
	}
	c, err := llm.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient(%s): %v", model, err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// interactionsClient points a "gemini-interactions" client at the fake; the
// adapter appends /interactions to the API root.
func interactionsClient(t *testing.T, f *fakeAPI, model string, mods ...func(*llm.Config)) llm.Client {
	t.Helper()
	cfg := llm.Config{
		Provider: "gemini-interactions", Model: model, APIKey: fakeKey,
		BaseURL: f.srv.URL + "/v1beta", DisableRetries: true,
	}
	for _, m := range mods {
		m(&cfg)
	}
	c, err := llm.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient(%s): %v", model, err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func userReq(text string) llm.Request {
	return llm.Request{Messages: []llm.Message{llm.NewTextMessage(llm.RoleUser, text)}, MaxTokens: 1000}
}

// ---------------------------------------------------------------------------
// Wire builders
// ---------------------------------------------------------------------------

type usage struct{ In, Out, Think, Cached int }

// chunk builds one generateContent JSON object (also one SSE data payload).
// finish and token may be empty; u may be nil (usage omitted).
func chunk(text, finish, token string, u *usage) string {
	cand := map[string]any{"content": map[string]any{"role": "model", "parts": []any{map[string]any{"text": text}}}}
	if finish != "" {
		cand["finishReason"] = finish
	}
	if token != "" {
		cand["continuationToken"] = token
	}
	m := map[string]any{"candidates": []any{cand}}
	if u != nil {
		m["usageMetadata"] = map[string]any{
			"promptTokenCount": u.In, "candidatesTokenCount": u.Out,
			"thoughtsTokenCount": u.Think, "cachedContentTokenCount": u.Cached,
		}
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// callChunk is a chunk whose only part is a functionCall.
func callChunk(name string, args map[string]any, finish, token string) string {
	cand := map[string]any{"content": map[string]any{"role": "model", "parts": []any{
		map[string]any{"functionCall": map[string]any{"name": name, "args": args}},
	}}}
	if finish != "" {
		cand["finishReason"] = finish
	}
	if token != "" {
		cand["continuationToken"] = token
	}
	b, _ := json.Marshal(map[string]any{"candidates": []any{cand}})
	return string(b)
}

func sse(payloads ...string) string {
	var b strings.Builder
	for _, p := range payloads {
		fmt.Fprintf(&b, "data: %s\n\n", p)
	}
	return b.String()
}

// rp answers as SSE when stream is set, as plain JSON otherwise.
func rp(stream bool, payload string) reply {
	if stream {
		return sseReply(payload)
	}
	return jsonReply(payload)
}

func jsonReply(body string) reply       { return reply{Body: body} }
func sseReply(p ...string) reply        { return reply{Body: sse(p...), SSE: true} }
func statusReply(s int, b string) reply { return reply{Status: s, Body: b} }

// dig walks nested map[string]any by key.
func dig(m map[string]any, path ...string) any {
	var cur any = m
	for _, k := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	return cur
}

// asJSON renders a decoded value canonically, for equality checks.
func asJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// streamResult is a drained stream.
type streamResult struct {
	Events   []llm.StreamEvent
	Text     string
	Thinking string
	Tools    []llm.ToolCall
	Dones    []llm.StreamEvent
	Err      error
}

// drain reads a stream to its end or its first error.
func drain(t *testing.T, c llm.Client, req llm.Request) streamResult {
	t.Helper()
	var r streamResult
	for ev, err := range c.Stream(t.Context(), req) {
		if err != nil {
			r.Err = err
			return r
		}
		r.Events = append(r.Events, ev)
		switch ev.Type {
		case llm.EventContent:
			r.Text += ev.Text
		case llm.EventThinking:
			r.Thinking += ev.Thinking
		case llm.EventToolUse:
			r.Tools = append(r.Tools, *ev.ToolCall)
		case llm.EventDone:
			r.Dones = append(r.Dones, ev)
		}
	}
	return r
}
