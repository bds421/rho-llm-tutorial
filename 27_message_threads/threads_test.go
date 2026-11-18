package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
	"github.com/bds421/rho-llm/provider/anthropic"
)

// --- helpers ---------------------------------------------------------------

func fakeClient(t *testing.T, f *fakeAnthropic, mutate ...func(*llm.Config)) llm.Client {
	t.Helper()
	cfg := llm.Config{
		Provider: "anthropic", Model: "claude-haiku-5-5", APIKey: "test-key",
		BaseURL: f.srv.URL, MaxTokens: 64, Timeout: 10 * time.Second,
		DisableRetries: true, // a request the fake sees is a request the Session made
	}
	for _, m := range mutate {
		m(&cfg)
	}
	c, err := llm.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func threadSession(c llm.Client, opts ...llm.SessionOption) *llm.Session {
	base := []llm.SessionOption{llm.WithSystem("sys-v1"), llm.WithTools(noopTool), llm.WithMessageThreads()}
	return llm.NewSession(c, append(base, opts...)...)
}

func send(t *testing.T, s *llm.Session, prompt string) *llm.Response {
	t.Helper()
	resp, err := s.Send(context.Background(), prompt)
	if err != nil {
		t.Fatalf("Send(%q): %v", prompt, err)
	}
	return resp
}

func stream(t *testing.T, s *llm.Session, prompt string) *llm.Response {
	t.Helper()
	resp, err := llm.CollectStream(s.Stream(context.Background(), prompt))
	if err != nil {
		t.Fatalf("Stream(%q): %v", prompt, err)
	}
	return resp
}

// wantCall asserts the shape of request n (1-based) at the fake.
func wantCall(t *testing.T, f *fakeAnthropic, n int, typ, prev string, messages int, systemAndTools bool) {
	t.Helper()
	calls := f.calls()
	if len(calls) < n {
		t.Fatalf("only %d requests, wanted request %d", len(calls), n)
	}
	r := calls[n-1]
	gotTyp, gotPrev := r.thread()
	if gotTyp != typ || gotPrev != prev {
		t.Errorf("request %d: thread = (%q, %q), want (%q, %q)", n, gotTyp, gotPrev, typ, prev)
	}
	if r.nMessages() != messages {
		t.Errorf("request %d: %d messages, want %d", n, r.nMessages(), messages)
	}
	if r.has("system") != systemAndTools || r.has("tools") != systemAndTools {
		t.Errorf("request %d: system=%v tools=%v, want both %v", n, r.has("system"), r.has("tools"), systemAndTools)
	}
	if typ != "" && !strings.Contains(r.Beta, anthropic.MessageThreadsBeta) {
		t.Errorf("request %d: anthropic-beta %q lacks %s", n, r.Beta, anthropic.MessageThreadsBeta)
	}
	if typ == "" && strings.Contains(r.Beta, anthropic.MessageThreadsBeta) {
		t.Errorf("request %d: unthreaded request carries the thread beta: %q", n, r.Beta)
	}
}

// --- the happy path ----------------------------------------------------------

func TestThreadsSendOnlyTheDelta(t *testing.T) {
	f := newFakeAnthropic(t)
	s := threadSession(fakeClient(t, f))
	r1 := send(t, s, "one")
	r2 := send(t, s, "two")
	r3 := send(t, s, "three")

	wantCall(t, f, 1, "create", "", 1, true)
	wantCall(t, f, 2, "continue", r1.ID, 1, false)
	wantCall(t, f, 3, "continue", r2.ID, 1, false)
	if r1.ID == "" || r1.ID == r2.ID || r2.ID == r3.ID {
		t.Fatalf("IDs must be distinct and non-empty: %q %q %q", r1.ID, r2.ID, r3.ID)
	}
	if n := len(s.Conversation().Messages); n != 6 {
		t.Fatalf("transcript has %d messages, want 6 (the Session keeps the whole conversation)", n)
	}
	assertMirror(t, f, s, r3)
	if f.rejected != 0 {
		t.Errorf("server rejected %d requests", f.rejected)
	}
}

func TestThreadsAreOffByDefault(t *testing.T) {
	f := newFakeAnthropic(t)
	s := llm.NewSession(fakeClient(t, f), llm.WithSystem("sys-v1"), llm.WithTools(noopTool))
	send(t, s, "one")
	send(t, s, "two")
	wantCall(t, f, 1, "", "", 1, true)
	wantCall(t, f, 2, "", "", 3, true)
}

func TestThreadsStreamAndSendShareOneThread(t *testing.T) {
	f := newFakeAnthropic(t)
	s := threadSession(fakeClient(t, f))
	r1 := stream(t, s, "one")
	r2 := send(t, s, "two")
	r3 := stream(t, s, "three")
	wantCall(t, f, 1, "create", "", 1, true)
	wantCall(t, f, 2, "continue", r1.ID, 1, false)
	wantCall(t, f, 3, "continue", r2.ID, 1, false)
	assertMirror(t, f, s, r3)
}

func TestThreadsToolLoopSendsOnlyTheResult(t *testing.T) {
	f := newFakeAnthropic(t)
	f.hook = func(n int, _ recordedReq) *fakeReply {
		if n == 1 {
			return &fakeReply{ToolUse: true}
		}
		return nil
	}
	s := threadSession(fakeClient(t, f))
	r1 := send(t, s, "use the tool")
	if len(r1.ToolCalls) != 1 {
		t.Fatalf("expected a tool call, got %+v", r1)
	}
	r2, err := s.SendMessages(context.Background(), llm.NewToolResultMessage(r1.ToolCalls[0].ID, "done", false))
	if err != nil {
		t.Fatal(err)
	}
	wantCall(t, f, 2, "continue", r1.ID, 1, false)
	assertMirror(t, f, s, r2)
}

// --- fallbacks ---------------------------------------------------------------

func TestThreadsContinueErrorsReplayOnceInFull(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"thread_not_found", 404, threadErrBody("not_found_error", llm.ThreadNotFound)},
		{"thread_already_continued", 400, threadErrBody("invalid_request_error", llm.ThreadAlreadyContinued)},
		{"thread_continue_in_flight", 400, threadErrBody("invalid_request_error", llm.ThreadContinueInFlight)},
		{"thread_fingerprint_mismatch", 400, threadErrBody("invalid_request_error", llm.ThreadFingerprint)},
		{"bare 400, no code", 400, plainErrBody("invalid_request_error", "cannot parse continuation")},
		{"bare 404, no code", 404, plainErrBody("not_found_error", "no such thing")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeAnthropic(t)
			f.hook = func(n int, _ recordedReq) *fakeReply {
				if n == 2 { // the first continuation
					return &fakeReply{Status: tc.status, Body: tc.body}
				}
				return nil
			}
			s := threadSession(fakeClient(t, f))
			send(t, s, "one")
			r2 := send(t, s, "two") // must succeed through the replay
			r3 := send(t, s, "three")

			// 1 create, 2 failed continue, 3 full replay as a NEW thread, 4 continue on it.
			if n := len(f.calls()); n != 4 {
				t.Fatalf("%d requests, want exactly 4 (one replay)", n)
			}
			wantCall(t, f, 2, "continue", "msg_1", 1, false)
			wantCall(t, f, 3, "create", "", 3, true)
			wantCall(t, f, 4, "continue", r2.ID, 1, false)
			if r2.Content != "ack two" {
				t.Errorf("reply %q", r2.Content)
			}
			assertMirror(t, f, s, r3)
			if n := len(s.Conversation().Messages); n != 6 {
				t.Errorf("transcript has %d messages, want 6 (replay must not duplicate)", n)
			}
		})
	}
}

func TestThreadsReplayThatFailsReturnsItsError(t *testing.T) {
	f := newFakeAnthropic(t)
	f.hook = func(n int, _ recordedReq) *fakeReply {
		switch n {
		case 2:
			return &fakeReply{Status: 404, Body: threadErrBody("not_found_error", llm.ThreadNotFound)}
		case 3: // the replay hits an unrelated error
			return &fakeReply{Status: 401, Body: plainErrBody("authentication_error", "bad key")}
		}
		return nil
	}
	s := threadSession(fakeClient(t, f))
	send(t, s, "one")
	_, err := s.Send(context.Background(), "two")
	var apiErr *llm.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 401 {
		t.Fatalf("want the replay's 401, got %v", err)
	}
	if n := len(f.calls()); n != 3 {
		t.Fatalf("%d requests: a failed replay must not loop", n)
	}
	if n := len(s.Conversation().Messages); n != 2 {
		t.Fatalf("failed turn left %d messages, want 2", n)
	}
	// The thread was dropped by the failed continue, so the next turn is a full create.
	send(t, s, "two again")
	wantCall(t, f, 4, "create", "", 3, true)
}

func TestThreadsUnsupportedTurnsThreadsOff(t *testing.T) {
	f := newFakeAnthropic(t)
	f.hook = func(n int, _ recordedReq) *fakeReply {
		if n == 1 { // the very first create is refused
			return &fakeReply{Status: 400, Body: threadErrBody("invalid_request_error", llm.ThreadUnsupported)}
		}
		return nil
	}
	s := threadSession(fakeClient(t, f))
	send(t, s, "one")
	send(t, s, "two")
	send(t, s, "three")
	wantCall(t, f, 1, "create", "", 1, true)
	wantCall(t, f, 2, "", "", 1, true) // plain replay of turn 1
	wantCall(t, f, 3, "", "", 3, true) // threads stay off: full history, no thread
	wantCall(t, f, 4, "", "", 5, true)
	if n := len(f.calls()); n != 4 {
		t.Fatalf("%d requests, want 4", n)
	}
}

func TestThreadsUnsupportedOnTheReplayOfAContinue(t *testing.T) {
	f := newFakeAnthropic(t)
	f.hook = func(n int, _ recordedReq) *fakeReply {
		switch n {
		case 2:
			return &fakeReply{Status: 404, Body: threadErrBody("not_found_error", llm.ThreadNotFound)}
		case 3: // the new-thread replay is refused too
			return &fakeReply{Status: 400, Body: threadErrBody("invalid_request_error", llm.ThreadUnsupported)}
		}
		return nil
	}
	s := threadSession(fakeClient(t, f))
	send(t, s, "one")
	send(t, s, "two")
	send(t, s, "three")
	wantCall(t, f, 2, "continue", "msg_1", 1, false)
	wantCall(t, f, 3, "create", "", 3, true)
	wantCall(t, f, 4, "", "", 3, true) // plain
	wantCall(t, f, 5, "", "", 5, true) // and stays plain
}

func TestThreadsUnrelatedErrorsAreNotThreadFallbacks(t *testing.T) {
	for _, status := range []int{401, 403, 413, 429, 500, 529} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			f := newFakeAnthropic(t)
			f.hook = func(n int, _ recordedReq) *fakeReply {
				if n == 2 {
					return &fakeReply{Status: status, Body: plainErrBody("api_error", "boom")}
				}
				return nil
			}
			s := threadSession(fakeClient(t, f)) // DisableRetries: one attempt per turn
			r1 := send(t, s, "one")
			if _, err := s.Send(context.Background(), "two"); err == nil {
				t.Fatal("expected the error to surface")
			}
			if n := len(f.calls()); n != 2 {
				t.Fatalf("%d requests: the error must not be replayed as a thread fallback", n)
			}
			if n := len(s.Conversation().Messages); n != 2 {
				t.Fatalf("failed turn left %d messages", n)
			}
			// The thread survives: the next turn continues from the same tip.
			r3 := send(t, s, "two again")
			wantCall(t, f, 3, "continue", r1.ID, 1, false)
			assertMirror(t, f, s, r3)
		})
	}
}

// With the library's own retries on, a 500 is retried by the pool - and the
// retry is the same continuation, not a full replay.
func TestThreadsServerErrorRetriedByThePoolKeepsTheThread(t *testing.T) {
	f := newFakeAnthropic(t)
	f.hook = func(n int, _ recordedReq) *fakeReply {
		if n == 2 {
			return &fakeReply{Status: 500, Body: plainErrBody("api_error", "transient")}
		}
		return nil
	}
	s := threadSession(fakeClient(t, f, func(c *llm.Config) {
		c.DisableRetries = false
		c.MaxRetries = 3
		c.RetryPolicy = &llm.RetryPolicy{BaseDelay: time.Millisecond, MaxDelay: 2 * time.Millisecond, Factor: 1}
	}))
	r1 := send(t, s, "one")
	r2 := send(t, s, "two")
	if n := len(f.calls()); n != 3 {
		t.Fatalf("%d requests, want 3 (create, failed continue, retried continue)", n)
	}
	wantCall(t, f, 2, "continue", r1.ID, 1, false)
	wantCall(t, f, 3, "continue", r1.ID, 1, false)
	assertMirror(t, f, s, r2)
}

func TestThreadsContextCancelIsNotReplayed(t *testing.T) {
	f := newFakeAnthropic(t)
	s := threadSession(fakeClient(t, f))
	send(t, s, "one")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Send(ctx, "two"); err == nil {
		t.Fatal("expected a cancellation error")
	}
	if n := len(s.Conversation().Messages); n != 2 {
		t.Fatalf("transcript has %d messages after a cancelled turn", n)
	}
	if n := len(f.calls()); n > 2 {
		t.Fatalf("%d requests: a cancelled turn must not be replayed", n)
	}
}

func TestThreadsStreamNotFoundBeforeEventsIsReplayed(t *testing.T) {
	f := newFakeAnthropic(t)
	f.hook = func(n int, _ recordedReq) *fakeReply {
		if n == 2 {
			return &fakeReply{Status: 404, Body: threadErrBody("not_found_error", llm.ThreadNotFound)}
		}
		return nil
	}
	s := threadSession(fakeClient(t, f))
	send(t, s, "one")
	r2 := stream(t, s, "two")
	wantCall(t, f, 3, "create", "", 3, true)
	if r2.Content != "ack two" {
		t.Errorf("streamed reply %q", r2.Content)
	}
	assertMirror(t, f, s, r2)
}

// --- changed settings --------------------------------------------------------

// A system message appended mid-session changes the system prompt the thread
// was created with, so the next turn must be a full request on a NEW thread.
func TestThreadsChangedSystemPromptStartsANewThread(t *testing.T) {
	f := newFakeAnthropic(t)
	s := threadSession(fakeClient(t, f))
	send(t, s, "one")
	s.Append(llm.NewSystemMessage("new rules"))
	r2 := send(t, s, "two")
	req := f.calls()[1]
	if typ, _ := req.thread(); typ != "create" {
		t.Fatalf("a changed system prompt must create a new thread, got %q", typ)
	}
	if req.nMessages() < 3 {
		t.Errorf("new thread must carry the history, got %d messages", req.nMessages())
	}
	if sys := blockText(req.Body["system"]); !strings.Contains(sys, "new rules") {
		t.Errorf("system %q lacks the new rules", sys)
	}
	r3 := send(t, s, "three")
	wantCall(t, f, 3, "continue", r2.ID, 1, false)
	assertMirror(t, f, s, r3)
}

// A conversation resumed with edited system/tools is a new Session and never
// reuses a thread: thread state is not part of the serialized Conversation.
func TestThreadsResumedConversationCreatesANewThread(t *testing.T) {
	f := newFakeAnthropic(t)
	s := threadSession(fakeClient(t, f))
	send(t, s, "one")
	send(t, s, "two")

	data, err := json.Marshal(s.Conversation())
	if err != nil {
		t.Fatal(err)
	}
	restored, err := llm.LoadConversation(data)
	if err != nil {
		t.Fatal(err)
	}
	restored.System = "sys-v2" // edited while "on disk"
	s2 := llm.NewSession(fakeClient(t, f), llm.WithConversation(restored), llm.WithMessageThreads())
	r := send(t, s2, "three")
	wantCall(t, f, 3, "create", "", 5, true)
	assertMirror(t, f, s2, r)
}

func TestThreadsModelChangeViaSwitchProviderStartsANewThread(t *testing.T) {
	f := newFakeAnthropic(t)
	s := threadSession(fakeClient(t, f))
	send(t, s, "one")
	send(t, s, "two")
	other := fakeClient(t, f, func(c *llm.Config) { c.Model = "claude-sonnet-5-5" })
	s.SwitchProvider(other)
	r := send(t, s, "three")
	wantCall(t, f, 3, "create", "", 5, true)
	if got := f.calls()[2].Body["model"]; got != "claude-sonnet-5-5" {
		t.Errorf("model = %v", got)
	}
	assertMirror(t, f, s, r)
}

func TestThreadsSwitchProviderDropsTheThreadEvenToTheSameClient(t *testing.T) {
	f := newFakeAnthropic(t)
	c := fakeClient(t, f)
	s := threadSession(c)
	send(t, s, "one")
	s.SwitchProvider(c)
	r := send(t, s, "two")
	wantCall(t, f, 2, "create", "", 3, true)
	assertMirror(t, f, s, r)
}

// --- non-Anthropic clients ---------------------------------------------------

func newCompatFake(t *testing.T) (*httptest.Server, *[]map[string]any, *sync.Mutex) {
	t.Helper()
	var mu sync.Mutex
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var b map[string]any
		_ = json.Unmarshal(raw, &b)
		mu.Lock()
		bodies = append(bodies, b)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"chatcmpl-1","object":"chat.completion","model":"llama-3.3-70b-versatile","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`)
	}))
	t.Cleanup(srv.Close)
	return srv, &bodies, &mu
}

func compatClient(t *testing.T, url string) llm.Client {
	t.Helper()
	c, err := llm.NewClient(llm.Config{Provider: "groq", Model: "llama-3.3-70b-versatile", APIKey: "k", BaseURL: url, MaxTokens: 32, DisableRetries: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestThreadsAreInertOnANonAnthropicSession(t *testing.T) {
	srv, bodies, mu := newCompatFake(t)
	s := llm.NewSession(compatClient(t, srv.URL), llm.WithMessageThreads(), llm.WithSystem("sys"))
	send(t, s, "one")
	send(t, s, "two")
	mu.Lock()
	defer mu.Unlock()
	if len(*bodies) != 2 {
		t.Fatalf("%d requests", len(*bodies))
	}
	for i, b := range *bodies {
		if _, ok := b["thread"]; ok {
			t.Errorf("request %d carries a thread to a non-Anthropic host", i+1)
		}
	}
	if m, _ := (*bodies)[1]["messages"].([]any); len(m) != 4 { // system + 3 history messages
		t.Errorf("second request carried %d messages, want the full history (4)", len(m))
	}
}

func TestDirectThreadOnANonAnthropicClientIsRefusedBeforeTheNetwork(t *testing.T) {
	srv, bodies, mu := newCompatFake(t)
	c := compatClient(t, srv.URL)
	_, err := c.Complete(context.Background(), llm.Request{
		Messages: []llm.Message{llm.NewTextMessage(llm.RoleUser, "hi")},
		Thread:   &llm.Thread{},
	})
	if err == nil || !strings.Contains(err.Error(), "message threads") {
		t.Fatalf("want a refusal naming message threads, got %v", err)
	}
	// Stream refuses too.
	var sawErr error
	for _, e := range c.Stream(context.Background(), llm.Request{
		Messages: []llm.Message{llm.NewTextMessage(llm.RoleUser, "hi")},
		Thread:   &llm.Thread{},
	}) {
		if e != nil {
			sawErr = e
		}
	}
	if sawErr == nil {
		t.Fatal("Stream accepted a Thread on a non-Anthropic client")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*bodies) != 0 {
		t.Fatalf("refusal must happen before any request, saw %d", len(*bodies))
	}
}

func TestSwitchingToAnotherProviderAndBackNeverContinuesAStaleThread(t *testing.T) {
	f := newFakeAnthropic(t)
	srv, bodies, mu := newCompatFake(t)
	anth := fakeClient(t, f)
	s := threadSession(anth)
	send(t, s, "one")
	s.SwitchProvider(compatClient(t, srv.URL))
	send(t, s, "two")
	mu.Lock()
	if _, ok := (*bodies)[0]["thread"]; ok {
		t.Error("compat host got a thread")
	}
	mu.Unlock()
	s.SwitchProvider(anth)
	r := send(t, s, "three")
	last := f.calls()[len(f.calls())-1]
	if typ, _ := last.thread(); typ != "create" {
		t.Fatalf("after a round trip the Session must create a new thread, got %q", typ)
	}
	assertMirror(t, f, s, r)
}

// --- the transcript is the same with or without threads -----------------------

func TestConversationJSONIsIdenticalWithAndWithoutThreads(t *testing.T) {
	run := func(threads bool) []byte {
		f := newFakeAnthropic(t)
		opts := []llm.SessionOption{llm.WithSystem("sys-v1"), llm.WithTools(noopTool)}
		if threads {
			opts = append(opts, llm.WithMessageThreads())
		}
		s := llm.NewSession(fakeClient(t, f), opts...)
		send(t, s, "one")
		stream(t, s, "two")
		send(t, s, "three")
		b, err := json.Marshal(s.Conversation())
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	// The fake numbers IDs per instance, so both runs see msg_1.. in order and
	// the transcripts are comparable byte for byte.
	on, off := run(true), run(false)
	if string(on) != string(off) {
		t.Fatalf("serialized conversation differs\non:  %s\noff: %s", on, off)
	}
	if strings.Contains(string(on), "thread") || strings.Contains(string(on), "previous_message_id") {
		t.Errorf("thread state leaked into the Conversation JSON: %s", on)
	}
	if _, err := llm.LoadConversation(on); err != nil {
		t.Errorf("round trip: %v", err)
	}
}

// --- hostile server answers --------------------------------------------------

func TestThreadsResponseWithoutAnIDEndsTheThread(t *testing.T) {
	f := newFakeAnthropic(t)
	f.hook = func(n int, _ recordedReq) *fakeReply {
		if n == 2 {
			return &fakeReply{NoID: true}
		}
		return nil
	}
	s := threadSession(fakeClient(t, f))
	send(t, s, "one")
	send(t, s, "two") // reply has no ID: nothing to continue from
	r3 := send(t, s, "three")
	wantCall(t, f, 3, "create", "", 5, true)
	r4 := send(t, s, "four")
	wantCall(t, f, 4, "continue", r3.ID, 1, false)
	assertMirror(t, f, s, r4)
}

func TestThreadsRefusalEndsTheThread(t *testing.T) {
	f := newFakeAnthropic(t)
	f.hook = func(n int, _ recordedReq) *fakeReply {
		if n == 2 {
			return &fakeReply{Refusal: true}
		}
		return nil
	}
	s := threadSession(fakeClient(t, f))
	send(t, s, "one")
	r2 := send(t, s, "two")
	if r2.StopReason != llm.StopRefusal {
		t.Fatalf("stop reason %q", r2.StopReason)
	}
	send(t, s, "three")
	wantCall(t, f, 3, "create", "", 5, true) // the API may hold the refused turn: start fresh
}

func TestThreadsBrokenJSONReplyDoesNotCorruptTheTranscript(t *testing.T) {
	f := newFakeAnthropic(t)
	f.hook = func(n int, _ recordedReq) *fakeReply {
		if n == 2 {
			return &fakeReply{BrokenJSON: true}
		}
		return nil
	}
	s := threadSession(fakeClient(t, f))
	r1 := send(t, s, "one")
	if _, err := s.Send(context.Background(), "two"); err == nil {
		t.Fatal("a truncated JSON body must be an error")
	}
	if n := len(s.Conversation().Messages); n != 2 {
		t.Fatalf("transcript has %d messages after a broken reply, want 2", n)
	}
	// The server consumed r1 as a continuation point (the reply was lost), so the
	// next continue is answered thread_already_continued and replayed in full.
	r3 := send(t, s, "two again")
	if n := len(f.calls()); n < 3 {
		t.Fatalf("%d requests", n)
	}
	assertMirror(t, f, s, r3)
	_ = r1
}

func TestThreadsStreamCutMidwayDoesNotCorruptTheTranscript(t *testing.T) {
	f := newFakeAnthropic(t)
	f.hook = func(n int, _ recordedReq) *fakeReply {
		if n == 2 {
			return &fakeReply{CutStream: true}
		}
		return nil
	}
	s := threadSession(fakeClient(t, f))
	send(t, s, "one")
	var events int
	for _, err := range s.Stream(context.Background(), "two") {
		_ = err
		events++
	}
	if n := len(s.Conversation().Messages); n != 2 {
		t.Fatalf("transcript has %d messages after a cut stream, want 2 (no partial turn)", n)
	}
	// Recovery: the next turn works and the server mirrors the transcript.
	r := send(t, s, "two again")
	assertMirror(t, f, s, r)
	if got := s.Conversation().Messages; len(got) != 4 || got[2].Content[0].Text != "two again" {
		t.Fatalf("transcript after recovery: %+v", got)
	}
}

func TestThreadsGarbageBodiesNeverPanic(t *testing.T) {
	for _, body := range []string{``, `null`, `[]`, `{"id":5}`, `{"id":"msg_x","content":"not an array"}`, `<html>502</html>`} {
		t.Run(fmt.Sprintf("%.12q", body), func(t *testing.T) {
			f := newFakeAnthropic(t)
			f.hook = func(n int, _ recordedReq) *fakeReply {
				if n == 2 {
					return &fakeReply{Status: 200, Body: body}
				}
				return nil
			}
			s := threadSession(fakeClient(t, f))
			send(t, s, "one")
			_, err := s.Send(context.Background(), "two")
			// Either a clean error or a (degenerate) success is fine; the
			// transcript must stay a whole number of turns.
			n := len(s.Conversation().Messages)
			if (err != nil && n != 2) || (err == nil && n != 4) {
				t.Fatalf("err=%v but transcript has %d messages", err, n)
			}
			// and the Session still works
			if _, err := s.Send(context.Background(), "three"); err != nil {
				t.Fatalf("session unusable after garbage reply: %v", err)
			}
		})
	}
}

// --- concurrency -------------------------------------------------------------

func TestThreadsConcurrentSendsOnOneSession(t *testing.T) {
	f := newFakeAnthropic(t)
	s := threadSession(fakeClient(t, f))
	const n = 8
	var wg sync.WaitGroup
	resps := make([]*llm.Response, n)
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				resps[i], errs[i] = s.Send(context.Background(), fmt.Sprintf("q%d", i))
			} else {
				resps[i], errs[i] = llm.CollectStream(s.Stream(context.Background(), fmt.Sprintf("q%d", i)))
			}
			_ = s.Conversation() // snapshots interleave with turns
			_ = s.Usage()
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("turn %d: %v", i, err)
		}
	}
	calls := f.calls()
	if len(calls) != n || f.rejected != 0 {
		t.Fatalf("%d requests, %d rejected; turns are serialized so each is exactly one clean request", len(calls), f.rejected)
	}
	prev := ""
	for i, c := range calls {
		typ, p := c.thread()
		if i == 0 && typ != "create" {
			t.Errorf("first request %q", typ)
		}
		if i > 0 && (typ != "continue" || p != prev) {
			t.Errorf("request %d: (%s, %s), want continue from %s", i+1, typ, p, prev)
		}
		prev = fmt.Sprintf("msg_%d", i+1)
	}
	conv := s.Conversation()
	if len(conv.Messages) != 2*n {
		t.Fatalf("%d messages, want %d", len(conv.Messages), 2*n)
	}
	for i, m := range conv.Messages {
		if want := []llm.Role{llm.RoleUser, llm.RoleAssistant}[i%2]; m.Role != want {
			t.Fatalf("message %d role %s, want %s: turns interleaved", i, m.Role, want)
		}
	}
	held, _ := f.threadMsgs(fmt.Sprintf("msg_%d", n))
	if strings.Join(held, "\n") != strings.Join(transcriptMirror(conv), "\n") {
		t.Fatalf("server thread diverged from transcript under concurrency")
	}
}

// --- direct clients and error codes -------------------------------------------

func TestDirectThreadCreateContinueAndErrors(t *testing.T) {
	f := newFakeAnthropic(t)
	c := fakeClient(t, f)
	ctx := context.Background()
	user := func(s string) []llm.Message { return []llm.Message{llm.NewTextMessage(llm.RoleUser, s)} }

	first, err := c.Complete(ctx, llm.Request{System: "sys", Messages: user("a"), Thread: &llm.Thread{}})
	if err != nil || first.ID == "" {
		t.Fatalf("create: %+v %v", first, err)
	}
	second, err := c.Complete(ctx, llm.Request{Messages: user("b"), Thread: &llm.Thread{PreviousMessageID: first.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if held, _ := f.threadMsgs(second.ID); len(held) != 4 {
		t.Errorf("server holds %q", held)
	}

	// Each ID continues once.
	_, err = c.Complete(ctx, llm.Request{Messages: user("c"), Thread: &llm.Thread{PreviousMessageID: first.ID}})
	if got := llm.ThreadErrorCode(err); got != llm.ThreadAlreadyContinued {
		t.Errorf("re-continue: code %q, err %v", got, err)
	}
	_, err = c.Complete(ctx, llm.Request{Messages: user("c"), Thread: &llm.Thread{PreviousMessageID: "msg_nope"}})
	if got := llm.ThreadErrorCode(err); got != llm.ThreadNotFound {
		t.Errorf("unknown ID: code %q, err %v", got, err)
	}

	// Request.System on a continuation is ignored on the wire (the thread keeps
	// the create request's; a Session relies on this), but a system MESSAGE in
	// the delta would change the prompt, so the adapter refuses it before any request.
	third, err := c.Complete(ctx, llm.Request{System: "ignored", Messages: user("sys-field"), Thread: &llm.Thread{PreviousMessageID: second.ID}})
	if err != nil {
		t.Fatalf("continue with Request.System set: %v", err)
	}
	if last := f.calls()[len(f.calls())-1]; last.has("system") {
		t.Error("a continuation put system on the wire")
	}
	before := len(f.calls())
	_, err = c.Complete(ctx, llm.Request{
		Messages: []llm.Message{llm.NewSystemMessage("new rules"), llm.NewTextMessage(llm.RoleUser, "d")},
		Thread:   &llm.Thread{PreviousMessageID: third.ID},
	})
	if err == nil || !strings.Contains(err.Error(), "system prompt") {
		t.Errorf("want a refusal about the system prompt, got %v", err)
	}
	if len(f.calls()) != before {
		t.Error("the refused continuation still hit the network")
	}
	// ...and with no thread at all there is no beta header.
	if _, err := c.Complete(ctx, llm.Request{Messages: user("plain")}); err != nil {
		t.Fatal(err)
	}
	if beta := f.calls()[len(f.calls())-1].Beta; strings.Contains(beta, anthropic.MessageThreadsBeta) {
		t.Errorf("plain request carries the thread beta: %q", beta)
	}
}

func TestThreadErrorCodeIsStrict(t *testing.T) {
	api := func(status int, msg string) error {
		return &llm.APIError{StatusCode: status, Provider: "anthropic", Message: msg}
	}
	withCode := threadErrBody("x", llm.ThreadNotFound)
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"plain error", errors.New("boom"), ""},
		{"404 with code", api(404, withCode), llm.ThreadNotFound},
		{"400 with code", api(400, withCode), llm.ThreadNotFound},
		{"wrapped", fmt.Errorf("outer: %w", api(400, withCode)), llm.ThreadNotFound},
		{"500 carrying the text", api(500, withCode), ""},
		{"429 carrying the text", api(429, withCode), ""},
		{"400 without code", api(400, plainErrBody("invalid_request_error", "nope")), ""},
		{"400 empty body", api(400, ""), ""},
		{"400 truncated mid-code", api(400, `{"error":{"details":{"error_code":"thread_no`), ""},
		{"non-thread code", api(400, `{"error_code":"rate_limited"}`), ""},
		{"extra whitespace", api(400, `{"details":{"error_code" :  "thread_continue_in_flight"}}`), llm.ThreadContinueInFlight},
	}
	for _, tc := range cases {
		if got := llm.ThreadErrorCode(tc.err); got != tc.want {
			t.Errorf("%s: ThreadErrorCode = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// The 4m30s reuse window is checked against Session's unexported clock, which a
// consumer cannot set, so expiry is covered only by the library-internal
// TestThreadsExpireByTheClock. From outside, expiry looks like thread_not_found,
// which TestThreadsContinueErrorsReplayOnceInFull covers.
