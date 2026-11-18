package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

// recorded is one request the fake server saw.
type recorded struct {
	Path string
	Auth string
	Body map[string]any
	Raw  string
}

// fake is an httptest server that records requests and answers with a canned
// body.
type fake struct {
	*httptest.Server
	mu   sync.Mutex
	reqs []recorded
	hits atomic.Int64
}

func newFake(t *testing.T, contentType, body string) *fake {
	t.Helper()
	f := &fake{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		raw, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		_ = json.Unmarshal(raw, &parsed)
		f.mu.Lock()
		f.reqs = append(f.reqs, recorded{Path: r.URL.Path, Auth: r.Header.Get("Authorization"), Body: parsed, Raw: string(raw)})
		f.mu.Unlock()
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fake) last(t *testing.T) recorded {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.reqs) == 0 {
		t.Fatal("fake server saw no request")
	}
	return f.reqs[len(f.reqs)-1]
}

func agentClient(t *testing.T, baseURL, model string) llm.Client {
	t.Helper()
	c, err := llm.NewClient(llm.Config{Provider: "perplexity-agent", Model: model, APIKey: "test-key", BaseURL: baseURL, MaxTokens: 100, MaxRetries: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func userReq(text string) llm.Request {
	return llm.Request{Messages: []llm.Message{llm.NewTextMessage(llm.RoleUser, text)}}
}

// sse builds a Server-Sent-Events body from raw data payloads.
func sse(events ...string) string {
	var b strings.Builder
	for _, e := range events {
		b.WriteString("data: " + e + "\n\n")
	}
	return b.String()
}

// streamResult is everything one Stream call produced.
type streamResult struct {
	text  string
	calls []llm.ToolCall
	done  *llm.StreamEvent
	errs  []error
}

func collect(c llm.Client, req llm.Request) streamResult {
	var r streamResult
	for ev, err := range c.Stream(context.Background(), req) {
		if err != nil {
			r.errs = append(r.errs, err)
			continue
		}
		switch ev.Type {
		case llm.EventContent:
			r.text += ev.Text
		case llm.EventToolUse:
			r.calls = append(r.calls, *ev.ToolCall)
		case llm.EventDone:
			e := ev
			r.done = &e
		}
	}
	return r
}

// Canned SSE events.
const (
	evCompletedEmpty = `{"type":"response.completed","response":{"status":"completed","output":[]}}`
)

func completedWithCall(id, callID, name, args string) string {
	b, _ := json.Marshal(map[string]any{"type": "response.completed", "response": map[string]any{
		"status": "completed",
		"output": []any{map[string]any{"type": "function_call", "id": id, "call_id": callID, "name": name, "arguments": args}},
	}})
	return string(b)
}

func itemDone(id, callID, name, args string) string {
	b, _ := json.Marshal(map[string]any{"type": "response.output_item.done", "output_index": 0, "item": map[string]any{
		"type": "function_call", "id": id, "call_id": callID, "name": name, "arguments": args,
	}})
	return string(b)
}

func newStatusServer(t *testing.T, code int) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"nope"}}`, code)
	}))
	t.Cleanup(s.Close)
	return s
}
