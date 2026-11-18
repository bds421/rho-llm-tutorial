package main

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/bds421/rho-llm"
)

const textDelta = `{"type":"response.output_text.delta","delta":"%s"}`

func delta(s string) string { return strings.Replace(textDelta, "%s", s, 1) }

func streamOf(t *testing.T, events ...string) streamResult {
	t.Helper()
	f := newFake(t, "text/event-stream", sse(events...))
	return collect(agentClient(t, f.URL, "perplexity/sonar"), userReq("go"))
}

func wantOneCall(t *testing.T, r streamResult, name, callID string) {
	t.Helper()
	if len(r.errs) != 0 {
		t.Fatalf("errors: %v", r.errs)
	}
	if len(r.calls) != 1 {
		t.Fatalf("calls = %d (%+v), want exactly 1", len(r.calls), r.calls)
	}
	if r.calls[0].Name != name || r.calls[0].ID != callID {
		t.Errorf("call = %+v", r.calls[0])
	}
	if r.done == nil || r.done.StopReason != llm.StopToolUse {
		t.Errorf("done = %+v, want tool_use stop", r.done)
	}
}

func TestStreamNormalDeltas(t *testing.T) {
	r := streamOf(t, delta("Hel"), delta("lo"),
		`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":50,"output_tokens":4,"input_tokens_details":{"cached_tokens":20}}}}`)
	if len(r.errs) != 0 || r.text != "Hello" || r.done == nil {
		t.Fatalf("text=%q done=%v errs=%v", r.text, r.done, r.errs)
	}
	if r.done.InputTokens != 30 || r.done.CacheReadTokens != 20 || r.done.OutputTokens != 4 {
		t.Errorf("usage = %+v", r.done)
	}
}

func TestStreamCallOnlyInOutputItemDone(t *testing.T) {
	r := streamOf(t, itemDone("fc_1", "call_1", "lookup", `{"q":"go"}`), evCompletedEmpty)
	wantOneCall(t, r, "lookup", "call_1")
	if m, _ := r.calls[0].Input.(map[string]any); m["q"] != "go" {
		t.Errorf("input = %v", r.calls[0].Input)
	}
}

func TestStreamCallOnlyInTerminalResponse(t *testing.T) {
	r := streamOf(t, completedWithCall("fc_1", "call_1", "lookup", `{"q":"go"}`))
	wantOneCall(t, r, "lookup", "call_1")
}

func TestStreamCallInDoneAndTerminalIsEmittedOnce(t *testing.T) {
	r := streamOf(t, itemDone("fc_1", "call_1", "lookup", `{"q":"go"}`),
		completedWithCall("fc_1", "call_1", "lookup", `{"q":"go"}`))
	wantOneCall(t, r, "lookup", "call_1")
}

func TestStreamCallInEveryPlaceIsEmittedOnce(t *testing.T) {
	r := streamOf(t,
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"lookup"}}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"{\"q\":"}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"\"go\"}"}`,
		`{"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0,"name":"lookup","call_id":"call_1","arguments":"{\"q\":\"go\"}"}`,
		itemDone("fc_1", "call_1", "lookup", `{"q":"go"}`),
		completedWithCall("fc_1", "call_1", "lookup", `{"q":"go"}`))
	wantOneCall(t, r, "lookup", "call_1")
}

// The item id differs between events but the call_id matches: still one call.
func TestStreamDedupByCallIDWhenItemIDsDiffer(t *testing.T) {
	r := streamOf(t, itemDone("fc_A", "call_1", "lookup", `{}`), completedWithCall("fc_B", "call_1", "lookup", `{}`))
	wantOneCall(t, r, "lookup", "call_1")
}

func TestStreamInterleavedCallsStaySeparate(t *testing.T) {
	r := streamOf(t,
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"a"}}`,
		`{"type":"response.output_item.added","output_index":1,"item":{"type":"function_call","id":"fc_2","call_id":"call_2","name":"b"}}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"{\"x\":"}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_2","output_index":1,"delta":"{\"y\":"}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"1}"}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_2","output_index":1,"delta":"2}"}`,
		`{"type":"response.function_call_arguments.done","item_id":"fc_2","output_index":1}`,
		`{"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0}`,
		evCompletedEmpty)
	if len(r.errs) != 0 || len(r.calls) != 2 {
		t.Fatalf("calls=%+v errs=%v", r.calls, r.errs)
	}
	got := map[string]any{}
	for _, c := range r.calls {
		got[c.Name] = c.Input
	}
	if a, _ := got["a"].(map[string]any); a["x"] != float64(1) {
		t.Errorf("a = %v", got["a"])
	}
	if b, _ := got["b"].(map[string]any); b["y"] != float64(2) {
		t.Errorf("b = %v (arguments spliced?)", got["b"])
	}
}

func TestStreamTwoCallsOnlyInTerminalResponse(t *testing.T) {
	r := streamOf(t, `{"type":"response.completed","response":{"status":"completed","output":[`+
		`{"type":"function_call","id":"fc_1","call_id":"c1","name":"a","arguments":"{}"},`+
		`{"type":"function_call","id":"fc_2","call_id":"c2","name":"b","arguments":"{}"}]}}`)
	if len(r.errs) != 0 || len(r.calls) != 2 || r.calls[0].Name != "a" || r.calls[1].Name != "b" {
		t.Fatalf("calls=%+v errs=%v", r.calls, r.errs)
	}
}

func TestStreamFailedWithoutNestedResponseIsError(t *testing.T) {
	for name, ev := range map[string]string{
		"top-level error": `{"type":"response.failed","error":{"code":"server_error","message":"agent exploded"}}`,
		"no error at all": `{"type":"response.failed"}`,
		"null response":   `{"type":"response.failed","response":null}`,
	} {
		r := streamOf(t, delta("partial"), ev)
		if r.done != nil {
			t.Errorf("%s: failed stream ended as a clean Done", name)
		}
		if len(r.errs) == 0 {
			t.Errorf("%s: no error surfaced", name)
			continue
		}
		if name == "top-level error" && !strings.Contains(r.errs[0].Error(), "agent exploded") {
			t.Errorf("%s: error lost its message: %v", name, r.errs[0])
		}
	}
}

func TestStreamFailedWithNestedResponseIsError(t *testing.T) {
	r := streamOf(t, `{"type":"response.failed","response":{"status":"failed","error":{"code":"x","message":"nested boom"},"usage":{"input_tokens":9,"output_tokens":1}}}`)
	if r.done != nil || len(r.errs) != 1 || !strings.Contains(r.errs[0].Error(), "nested boom") {
		t.Fatalf("done=%v errs=%v", r.done, r.errs)
	}
}

// A failed event whose nested status claims "completed" is still a failure.
func TestStreamFailedEventWithLyingStatusIsError(t *testing.T) {
	r := streamOf(t, `{"type":"response.failed","response":{"status":"completed"}}`)
	if r.done != nil || len(r.errs) == 0 {
		t.Fatalf("done=%v errs=%v", r.done, r.errs)
	}
}

func TestStreamErrorEvent(t *testing.T) {
	r := streamOf(t, `{"type":"error","error":{"code":"rate_limit","message":"slow down"}}`)
	if r.done != nil || len(r.errs) != 1 {
		t.Fatalf("done=%v errs=%v", r.done, r.errs)
	}
}

func TestStreamCutMidwayIsUnexpectedEOF(t *testing.T) {
	r := streamOf(t, delta("par"), delta("tial"), `{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"{\"q"}`)
	if r.done != nil || r.text != "partial" {
		t.Fatalf("text=%q done=%v", r.text, r.done)
	}
	if len(r.errs) != 1 || !errors.Is(r.errs[0], io.ErrUnexpectedEOF) {
		t.Fatalf("errs = %v, want ErrUnexpectedEOF", r.errs)
	}
	if len(r.calls) != 0 {
		t.Errorf("half-streamed call was emitted: %+v", r.calls)
	}
}

func TestStreamEmptyBodyIsError(t *testing.T) {
	r := streamOf(t)
	if r.done != nil || len(r.errs) == 0 {
		t.Fatalf("done=%v errs=%v", r.done, r.errs)
	}
}

func TestStreamGarbageLines(t *testing.T) {
	// Comments, blanks, non-data fields and unknown events are skipped.
	body := ": keep-alive\n\nevent: ping\nretry: 10\nnot an sse line\n\n" +
		"data: " + delta("ok") + "\n\n" +
		"data: " + `{"type":"response.something_new","x":1}` + "\n\n" +
		"data: " + evCompletedEmpty + "\n\n"
	f := newFake(t, "text/event-stream", body)
	r := collect(agentClient(t, f.URL, "perplexity/sonar"), userReq("go"))
	if r.text != "ok" || r.done == nil || len(r.errs) != 0 {
		t.Fatalf("text=%q done=%v errs=%v", r.text, r.done, r.errs)
	}
}

// A malformed data line is an error. The pooled client ends a stream at its
// first error (no retry mid-stream), so no Done follows.
func TestStreamMalformedEventEndsStreamWithError(t *testing.T) {
	body := "data: " + delta("ok") + "\n\ndata: {this is not json\n\ndata: " + evCompletedEmpty + "\n\n"
	f := newFake(t, "text/event-stream", body)
	r := collect(agentClient(t, f.URL, "perplexity/sonar"), userReq("go"))
	if r.text != "ok" || r.done != nil {
		t.Fatalf("text=%q done=%v", r.text, r.done)
	}
	if len(r.errs) != 1 || !strings.Contains(r.errs[0].Error(), "malformed") {
		t.Errorf("errs = %v, want one malformed-event error", r.errs)
	}
}

func TestStreamGarbageAfterTerminalDoesNotMaskCompletion(t *testing.T) {
	f := newFake(t, "text/event-stream", sse(delta("ok"), evCompletedEmpty)+"data: {garbage\n\n")
	r := collect(agentClient(t, f.URL, "perplexity/sonar"), userReq("go"))
	if r.done == nil || len(r.errs) != 0 {
		t.Fatalf("done=%v errs=%v", r.done, r.errs)
	}
}

func TestStreamHugeArgumentsWithinLimit(t *testing.T) {
	big := strings.Repeat("a", 512*1024)
	args := `{"blob":"` + big + `"}`
	r := streamOf(t, itemDone("fc_1", "call_1", "lookup", args), evCompletedEmpty)
	wantOneCall(t, r, "lookup", "call_1")
	if m, _ := r.calls[0].Input.(map[string]any); len(m["blob"].(string)) != len(big) {
		t.Error("huge argument truncated")
	}
}

func TestStreamArgumentsBeyondLimitAreRefused(t *testing.T) {
	const limit = 4096
	huge := strings.Repeat("a", limit*4)
	for name, events := range map[string][]string{
		"deltas":        {`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"` + huge + `"}`, evCompletedEmpty},
		"output_item":   {itemDone("fc_1", "call_1", "lookup", `{"b":"`+huge+`"}`), evCompletedEmpty},
		"terminal call": {completedWithCall("fc_1", "call_1", "lookup", `{"b":"`+huge+`"}`)},
	} {
		f := newFake(t, "text/event-stream", sse(events...))
		c, err := llm.NewClient(llm.Config{Provider: "perplexity-agent", Model: "perplexity/sonar", APIKey: "k", BaseURL: f.URL, MaxTokens: 10, MaxRetries: 1, MaxToolInputBytes: limit})
		if err != nil {
			t.Fatal(err)
		}
		r := collect(c, userReq("go"))
		c.Close()
		if len(r.calls) != 0 || len(r.errs) == 0 {
			t.Errorf("%s: calls=%d errs=%v, want a refusal", name, len(r.calls), r.errs)
		}
	}
}

func TestStreamMalformedToolArgumentsDoNotPanic(t *testing.T) {
	for _, args := range []string{"", "{", "null", "[1,2", `"str"`, "\x00\xff"} {
		r := streamOf(t, itemDone("fc_1", "call_1", "lookup", args), evCompletedEmpty)
		if len(r.calls) != 1 {
			t.Errorf("args %q: calls = %d", args, len(r.calls))
		}
	}
}

func TestStreamHTTPErrorIsAPIError(t *testing.T) {
	// A 4xx before any event: surfaced as one error, no events.
	c := agentClient(t, newStatusServer(t, 401).URL, "perplexity/sonar")
	r := collect(c, userReq("go"))
	if len(r.errs) != 1 || r.done != nil {
		t.Fatalf("errs=%v done=%v", r.errs, r.done)
	}
	var apiErr *llm.APIError
	if !errors.As(r.errs[0], &apiErr) || apiErr.StatusCode != 401 {
		t.Errorf("err = %v, want APIError 401", r.errs[0])
	}
	if strings.Contains(r.errs[0].Error(), "test-key") {
		t.Error("API key leaked into the error")
	}
}

func TestConcurrentStreamsStayIsolated(t *testing.T) {
	f := newFake(t, "text/event-stream", sse(delta("x"),
		itemDone("fc_1", "call_1", "lookup", `{"q":"go"}`), evCompletedEmpty))
	c := agentClient(t, f.URL, "perplexity/sonar")
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := collect(c, userReq("go"))
			if len(r.errs) != 0 || len(r.calls) != 1 || r.text != "x" || r.done == nil {
				t.Errorf("text=%q calls=%d errs=%v", r.text, len(r.calls), r.errs)
			}
		}()
	}
	wg.Wait()
	if got := f.hits.Load(); got != 32 {
		t.Errorf("hits = %d", got)
	}
}

func TestStreamConsumerStopsEarly(t *testing.T) {
	f := newFake(t, "text/event-stream", sse(delta("a"), delta("b"), delta("c"), evCompletedEmpty))
	c := agentClient(t, f.URL, "perplexity/sonar")
	n := 0
	for _, err := range c.Stream(t.Context(), userReq("go")) {
		if err != nil {
			t.Fatal(err)
		}
		n++
		break
	}
	if n != 1 {
		t.Errorf("n = %d", n)
	}
}
