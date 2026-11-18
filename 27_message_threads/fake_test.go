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

// fakeAnthropic is an httptest stand-in for the Anthropic Messages API with
// message-thread semantics: "create" starts a thread from a full request,
// "continue" appends to the thread a response ID names, each ID continues once,
// continuations may not carry system or tools, and an unknown ID is a 404.
// It records every request so tests can see what travelled.

type recordedReq struct {
	N    int
	Beta string
	Body map[string]any
}

func (r recordedReq) thread() (typ, prev string) {
	th, _ := r.Body["thread"].(map[string]any)
	typ, _ = th["type"].(string)
	prev, _ = th["previous_message_id"].(string)
	return
}

func (r recordedReq) nMessages() int {
	m, _ := r.Body["messages"].([]any)
	return len(m)
}

func (r recordedReq) has(key string) bool { _, ok := r.Body[key]; return ok }

// fakeReply overrides the fake's normal behaviour for one request.
type fakeReply struct {
	Status      int    // non-zero: answer with this status and Body, nothing else
	Body        string // raw body for Status
	NoID        bool   // answer normally but omit the message ID (thread not stored)
	Refusal     bool   // answer normally with stop_reason "refusal"
	ToolUse     bool   // answer with a tool_use block (Complete only)
	CutStream   bool   // streams: send message_start and some text, then drop the connection
	BrokenJSON  bool   // non-streams: 200 with a truncated JSON body
	Model       string // model echoed in the reply
	TextOverrid string
}

type fakeThread struct {
	msgs      []string // "role:text|text" per message, as the server holds them
	system    string
	hasTools  bool
	continued bool
}

type fakeAnthropic struct {
	t   *testing.T
	srv *httptest.Server

	mu       sync.Mutex
	reqs     []recordedReq
	threads  map[string]*fakeThread
	next     int
	rejected int
	// hook, when set, may override the answer to request n (1-based).
	hook func(n int, r recordedReq) *fakeReply
}

func newFakeAnthropic(t *testing.T) *fakeAnthropic {
	t.Helper()
	f := &fakeAnthropic{t: t, threads: map[string]*fakeThread{}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeAnthropic) calls() []recordedReq {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedReq(nil), f.reqs...)
}

func threadErrBody(typ, code string) string {
	return fmt.Sprintf(`{"type":"error","error":{"type":%q,"message":"thread problem","details":{"error_code":%q}}}`, typ, code)
}

func plainErrBody(typ, msg string) string {
	return fmt.Sprintf(`{"type":"error","error":{"type":%q,"message":%q}}`, typ, msg)
}

// blockText renders a content value (string or block array) the way both the
// server mirror and the transcript mirror compare it.
func blockText(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case []any:
		var parts []string
		for _, b := range c {
			bm, _ := b.(map[string]any)
			switch bm["type"] {
			case "text":
				parts = append(parts, fmt.Sprint(bm["text"]))
			case "tool_use":
				parts = append(parts, "use:"+fmt.Sprint(bm["id"]))
			case "tool_result":
				parts = append(parts, "result:"+blockText(bm["content"]))
			}
		}
		return strings.Join(parts, "|")
	}
	return ""
}

func (f *fakeAnthropic) serve(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)

	f.mu.Lock()
	rec := recordedReq{N: len(f.reqs) + 1, Beta: r.Header.Get("anthropic-beta"), Body: body}
	f.reqs = append(f.reqs, rec)
	var ov *fakeReply
	if f.hook != nil {
		ov = f.hook(rec.N, rec)
	}
	if ov != nil && ov.Status != 0 {
		f.rejected++
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(ov.Status)
		_, _ = io.WriteString(w, ov.Body)
		return
	}
	if ov == nil {
		ov = &fakeReply{}
	}

	typ, prev := rec.thread()
	var msgs []string
	var system string
	hasTools := rec.has("tools")
	fail := func(status int, b string) {
		f.rejected++
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, b)
	}
	if rec.has("system") {
		system = blockText(body["system"])
	}
	switch typ {
	case "continue":
		th := f.threads[prev]
		switch {
		case th == nil:
			fail(404, threadErrBody("not_found_error", "thread_not_found"))
			return
		case th.continued:
			fail(400, threadErrBody("invalid_request_error", "thread_already_continued"))
			return
		case rec.has("system") || rec.has("tools"):
			fail(400, plainErrBody("invalid_request_error", "system and tools are fixed at thread creation"))
			return
		}
		th.continued = true
		msgs = append(msgs, th.msgs...)
		system, hasTools = th.system, th.hasTools
	case "create", "":
	default:
		fail(400, plainErrBody("invalid_request_error", "bad thread type"))
		return
	}
	ms, _ := body["messages"].([]any)
	lastUser := ""
	for _, m := range ms {
		mm, _ := m.(map[string]any)
		text := blockText(mm["content"])
		msgs = append(msgs, fmt.Sprintf("%v:%s", mm["role"], text))
		if mm["role"] == "user" {
			lastUser = text
		}
	}

	f.next++
	id := fmt.Sprintf("msg_%d", f.next)
	reply := "ack " + lastUser
	if ov.TextOverrid != "" {
		reply = ov.TextOverrid
	}
	stop := "end_turn"
	content := fmt.Sprintf(`[{"type":"text","text":%q}]`, reply)
	stored := "assistant:" + reply
	if ov.ToolUse {
		stop = "tool_use"
		content = `[{"type":"tool_use","id":"toolu_1","name":"noop","input":{}}]`
		stored = "assistant:use:toolu_1"
	}
	if ov.Refusal {
		stop = "refusal"
	}
	model := ov.Model
	if model == "" {
		model = fmt.Sprint(body["model"])
	}
	inTokens := (len(msgs) + 1) * 10
	jsonID := fmt.Sprintf("%q", id)
	if ov.NoID {
		jsonID = `""`
	}
	if typ != "" && !ov.NoID && !ov.BrokenJSON {
		f.threads[id] = &fakeThread{msgs: append(msgs, stored), system: system, hasTools: hasTools}
	}
	f.mu.Unlock()

	if stream, _ := body["stream"].(bool); stream {
		w.Header().Set("Content-Type", "text/event-stream")
		sse := func(ev, data string) { fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev, data) }
		sse("message_start", fmt.Sprintf(`{"type":"message_start","message":{"id":%s,"model":%q,"usage":{"input_tokens":%d}}}`, jsonID, model, inTokens))
		sse("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		if ov.CutStream {
			sse("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"par"}}`)
			if fl, ok := w.(http.Flusher); ok {
				fl.Flush()
			}
			panic(http.ErrAbortHandler) // drop the connection mid-stream
		}
		sse("content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":%q}}`, reply))
		sse("content_block_stop", `{"type":"content_block_stop","index":0}`)
		sse("message_delta", fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":%q},"usage":{"output_tokens":5}}`, stop))
		sse("message_stop", `{"type":"message_stop"}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	out := fmt.Sprintf(`{"id":%s,"type":"message","role":"assistant","model":%q,"stop_reason":%q,"usage":{"input_tokens":%d,"output_tokens":5},"content":%s}`,
		jsonID, model, stop, inTokens, content)
	if ov.BrokenJSON {
		out = out[:len(out)/2]
	}
	_, _ = io.WriteString(w, out)
}

// threadMsgs returns what the server holds for the thread tipped by id.
func (f *fakeAnthropic) threadMsgs(id string) ([]string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	th, ok := f.threads[id]
	if !ok {
		return nil, false
	}
	return append([]string(nil), th.msgs...), true
}

// transcriptMirror renders a conversation the way the server mirror renders a
// thread, so the two can be compared for equality.
func transcriptMirror(c *llm.Conversation) []string {
	var out []string
	for _, m := range c.Messages {
		if m.Role == llm.RoleSystem {
			continue // hoisted into the request's system field, not a thread message
		}
		var parts []string
		for _, p := range m.Content {
			switch {
			case p.ToolName != "":
				parts = append(parts, "use:"+p.ToolUseID)
			case p.ToolResultID != "":
				parts = append(parts, "result:"+p.ToolResultContent)
			case p.Text != "":
				parts = append(parts, p.Text)
			}
		}
		out = append(out, fmt.Sprintf("%s:%s", m.Role, strings.Join(parts, "|")))
	}
	return out
}

// assertMirror proves the server holds exactly the session's transcript: the
// delta protocol lost nothing and duplicated nothing.
func assertMirror(t *testing.T, f *fakeAnthropic, sess *llm.Session, last *llm.Response) {
	t.Helper()
	held, ok := f.threadMsgs(last.ID)
	if !ok {
		t.Fatalf("server holds no thread tipped by %q", last.ID)
	}
	want := transcriptMirror(sess.Conversation())
	if strings.Join(held, "\n") != strings.Join(want, "\n") {
		t.Fatalf("server thread diverged from the transcript\nserver:     %q\ntranscript: %q", held, want)
	}
}
