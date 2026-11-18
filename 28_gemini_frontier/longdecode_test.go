package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
)

// maxHops mirrors the adapter's documented bound on one long-decode turn.
const maxHops = 64

func longReq() llm.Request {
	req := userReq("prove it")
	req.System = "be rigorous"
	req.MaxTokens = 262144
	req.ThinkingLevel = llm.ThinkingHigh
	req.Tools = []llm.Tool{{Name: "lookup", Description: "d", InputSchema: map[string]any{"type": "object"}}}
	return req
}

// Three hops: every follow-up is the SAME request plus the latest token at the
// top level. No partial output is added to contents; text joins, usage sums.
func TestLongDecodeCompleteThreeHops(t *testing.T) {
	f := newFake(t,
		jsonReply(chunk("one ", "CONTINUATION", "T1", &usage{In: 100, Out: 5, Think: 30})),
		jsonReply(chunk("two ", "CONTINUATION", "T2", &usage{In: 100, Out: 6, Think: 20, Cached: 90})),
		jsonReply(chunk("three", "STOP", "", &usage{In: 100, Out: 7, Think: 10, Cached: 90})),
	)
	c := geminiClient(t, f, "gemini-4-carbon")
	resp, err := c.Complete(t.Context(), longReq())
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "one two three" || resp.StopReason != llm.StopEndTurn || resp.RawStopReason != "STOP" {
		t.Errorf("content %q stop %q raw %q", resp.Content, resp.StopReason, resp.RawStopReason)
	}
	// 300 prompt tokens in total, 180 of them cache reads.
	if resp.InputTokens != 120 || resp.CacheReadTokens != 180 || resp.OutputTokens != 18 || resp.ThinkingTokens != 60 {
		t.Errorf("usage in=%d cache=%d out=%d think=%d, want 120/180/18/60",
			resp.InputTokens, resp.CacheReadTokens, resp.OutputTokens, resp.ThinkingTokens)
	}
	if f.hits() != 3 {
		t.Fatalf("%d requests, want 3", f.hits())
	}
	if _, has := f.req(0).Body["continuationToken"]; has {
		t.Error("first hop sent a continuationToken")
	}
	if f.req(1).Body["continuationToken"] != "T1" || f.req(2).Body["continuationToken"] != "T2" {
		t.Errorf("tokens %v, %v: want T1 then T2 at the top level",
			f.req(1).Body["continuationToken"], f.req(2).Body["continuationToken"])
	}
	for i := 1; i < 3; i++ {
		for _, key := range []string{"contents", "generationConfig", "systemInstruction", "tools"} {
			if asJSON(f.req(0).Body[key]) != asJSON(f.req(i).Body[key]) {
				t.Errorf("hop %d changed %s:\n%s\n%s", i, key, asJSON(f.req(0).Body[key]), asJSON(f.req(i).Body[key]))
			}
		}
		if len(f.req(i).Body["contents"].([]any)) != 1 {
			t.Errorf("hop %d: contents grew: partial output must not be replayed", i)
		}
	}
	if dig(f.req(0).Body, "generationConfig", "thinkingConfig", "thinkingLevel") != "high" {
		t.Error("thinking level lost")
	}
}

// Streaming: events of every hop reach the consumer, a mid-hop checkpoint token
// is superseded by the final one, and exactly one EventDone carries the sums.
func TestLongDecodeStreamThreeHops(t *testing.T) {
	f := newFake(t,
		sseReply(
			chunk("a", "", "", &usage{In: 10, Out: 1}),
			chunk("b", "", "CHECKPOINT", nil),
			chunk("", "CONTINUATION", "T1", &usage{In: 10, Out: 2})),
		sseReply(chunk("c", "", "", nil), chunk("", "CONTINUATION", "T2", &usage{In: 10, Out: 3})),
		sseReply(callChunk("lookup", map[string]any{"q": "x"}, "", ""), chunk("d", "STOP", "", &usage{In: 10, Out: 4})),
	)
	c := geminiClient(t, f, "gemini-4-hydrogen")
	r := drain(t, c, longReq())
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.Text != "abcd" {
		t.Errorf("text %q, want abcd", r.Text)
	}
	if len(r.Dones) != 1 || r.Events[len(r.Events)-1].Type != llm.EventDone {
		t.Fatalf("%d EventDone, want exactly one and last", len(r.Dones))
	}
	if d := r.Dones[0]; d.StopReason != llm.StopToolUse || d.InputTokens != 30 || d.OutputTokens != 9 {
		t.Errorf("done %+v, want tool_use, input 30, output 9", d)
	}
	if len(r.Tools) != 1 || r.Tools[0].Name != "lookup" {
		t.Errorf("tools %+v", r.Tools)
	}
	if f.hits() != 3 || f.req(1).Body["continuationToken"] != "T1" || f.req(2).Body["continuationToken"] != "T2" {
		t.Errorf("hits %d tokens %v %v", f.hits(), f.req(1).Body["continuationToken"], f.req(2).Body["continuationToken"])
	}
	for i := range 3 {
		if f.req(i).RawQuery != "alt=sse" {
			t.Errorf("hop %d query %q", i, f.req(i).RawQuery)
		}
	}
}

// Tool calls from different hops keep distinct IDs.
func TestLongDecodeToolCallsAcrossHops(t *testing.T) {
	f := newFake(t,
		jsonReply(callChunk("lookup", map[string]any{"q": "a"}, "CONTINUATION", "T1")),
		jsonReply(callChunk("lookup", map[string]any{"q": "b"}, "STOP", "")),
	)
	c := geminiClient(t, f, "gemini-4-carbon")
	resp, err := c.Complete(t.Context(), longReq())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 2 || resp.ToolCalls[0].ID == resp.ToolCalls[1].ID || resp.StopReason != llm.StopToolUse {
		t.Errorf("calls %+v stop %q", resp.ToolCalls, resp.StopReason)
	}
}

// ---------------------------------------------------------------------------
// Hostile hop sequences, each for Complete and Stream
// ---------------------------------------------------------------------------

// outcome is what both entry points reduce to.
type outcome struct {
	Text    string
	Stop    string
	Raw     string
	In, Out int
	Err     error
}

func runBoth(t *testing.T, mk func(stream bool) *fakeAPI, model string, mod func(*llm.Request)) map[string]struct {
	F *fakeAPI
	O outcome
} {
	t.Helper()
	out := map[string]struct {
		F *fakeAPI
		O outcome
	}{}
	req := longReq()
	if mod != nil {
		mod(&req)
	}
	{
		f := mk(false)
		resp, err := geminiClient(t, f, model).Complete(t.Context(), req)
		o := outcome{Err: err}
		if resp != nil {
			o = outcome{Text: resp.Content, Stop: resp.StopReason, Raw: resp.RawStopReason, In: resp.InputTokens, Out: resp.OutputTokens, Err: err}
		}
		out["complete"] = struct {
			F *fakeAPI
			O outcome
		}{f, o}
	}
	{
		f := mk(true)
		r := drain(t, geminiClient(t, f, model), req)
		o := outcome{Text: r.Text, Err: r.Err}
		if len(r.Dones) > 0 {
			d := r.Dones[len(r.Dones)-1]
			o.Stop, o.Raw, o.In, o.Out = d.StopReason, d.RawStopReason, d.InputTokens, d.OutputTokens
		}
		out["stream"] = struct {
			F *fakeAPI
			O outcome
		}{f, o}
	}
	return out
}

// A token on a finished turn is not a reason to continue.
func TestLongDecodeTokenWithoutContinuation(t *testing.T) {
	res := runBoth(t, func(stream bool) *fakeAPI {
		return newFake(t, rp(stream, chunk("done", "STOP", "STRAY", nil)))
	}, "gemini-4-carbon", nil)
	for name, r := range res {
		if r.F.hits() != 1 || r.O.Err != nil || r.O.Stop != llm.StopEndTurn || r.O.Text != "done" {
			t.Errorf("%s: hits %d %+v, want one request, end_turn", name, r.F.hits(), r.O)
		}
	}
}

// CONTINUATION without a token cannot be continued: it is reported as a
// truncated turn (max_tokens), not retried forever and not as success.
func TestLongDecodeContinuationWithoutToken(t *testing.T) {
	res := runBoth(t, func(stream bool) *fakeAPI {
		return newFake(t, rp(stream, chunk("partial", "CONTINUATION", "", nil)))
	}, "gemini-4-carbon", nil)
	for name, r := range res {
		if r.F.hits() != 1 || r.O.Err != nil || r.O.Stop != llm.StopMaxTokens || r.O.Raw != "CONTINUATION" || r.O.Text != "partial" {
			t.Errorf("%s: hits %d %+v, want 1 request, max_tokens/CONTINUATION", name, r.F.hits(), r.O)
		}
	}
}

// A server that answers CONTINUATION forever is cut off at the hop bound.
func TestLongDecodeEndlessContinuationIsBounded(t *testing.T) {
	res := runBoth(t, func(stream bool) *fakeAPI {
		return newFake(t, rp(stream, chunk("x", "CONTINUATION", "again", &usage{In: 1, Out: 1})))
	}, "gemini-4-hydrogen", nil)
	for name, r := range res {
		if r.F.hits() != maxHops {
			t.Errorf("%s: %d requests, want the bound %d", name, r.F.hits(), maxHops)
		}
		if r.O.Err != nil || r.O.Stop != llm.StopMaxTokens || r.O.Raw != "CONTINUATION" {
			t.Errorf("%s: %+v, want max_tokens/CONTINUATION", name, r.O)
		}
		if r.O.Text != strings.Repeat("x", maxHops) {
			t.Errorf("%s: %d bytes of text, want %d", name, len(r.O.Text), maxHops)
		}
		if r.O.In != maxHops || r.O.Out != maxHops {
			t.Errorf("%s: usage %d/%d, want %d/%d summed over every hop", name, r.O.In, r.O.Out, maxHops, maxHops)
		}
	}
}

// A later hop that fails fails the turn: never a half answer returned as success.
func TestLongDecodeLaterHopHTTPError(t *testing.T) {
	first := chunk("x", "CONTINUATION", "T1", nil)
	f := newFake(t, jsonReply(first), statusReply(http.StatusInternalServerError, `{"error":{"code":500,"message":"boom","status":"INTERNAL"}}`))
	resp, err := geminiClient(t, f, "gemini-4-carbon").Complete(t.Context(), longReq())
	var apiErr *llm.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 500 || resp != nil {
		t.Fatalf("resp %v err %v, want an APIError 500 and no response", resp, err)
	}
	if strings.Contains(err.Error(), fakeKey) {
		t.Error("error leaked the API key")
	}

	f = newFake(t, sseReply(chunk("x", "CONTINUATION", "T1", nil)), statusReply(http.StatusBadRequest, `{"error":{"code":400,"message":"bad token","status":"INVALID_ARGUMENT"}}`))
	r := drain(t, geminiClient(t, f, "gemini-4-carbon"), longReq())
	if !errors.As(r.Err, &apiErr) || apiErr.StatusCode != 400 {
		t.Errorf("stream err = %v, want an APIError 400", r.Err)
	}
	if r.Text != "x" || len(r.Dones) != 0 {
		t.Errorf("text %q dones %d: hop 1's events were already delivered and no Done may follow", r.Text, len(r.Dones))
	}
}

// Malformed JSON (or an in-band error object) on a later hop: an error.
func TestLongDecodeLaterHopMalformed(t *testing.T) {
	first := chunk("x", "CONTINUATION", "T1", nil)
	for name, body := range map[string]string{
		"truncated json": `{"candidates":[{"content":{"par`,
		"html":           `<html>502 Bad Gateway</html>`,
		"empty":          ``,
		"wrong type":     `{"candidates":"nope"}`,
	} {
		t.Run("complete/"+name, func(t *testing.T) {
			f := newFake(t, jsonReply(first), jsonReply(body))
			resp, err := geminiClient(t, f, "gemini-4-carbon").Complete(t.Context(), longReq())
			if err == nil {
				t.Fatalf("got %+v, want a decode error", resp)
			}
		})
	}
	t.Run("complete/in-band error", func(t *testing.T) {
		f := newFake(t, jsonReply(first), jsonReply(`{"error":{"code":429,"message":"quota","status":"RESOURCE_EXHAUSTED"}}`))
		_, err := geminiClient(t, f, "gemini-4-carbon").Complete(t.Context(), longReq())
		if err == nil {
			t.Fatal("want the in-band error")
		}
	})
	t.Run("stream/malformed event", func(t *testing.T) {
		f := newFake(t, sseReply(first), reply{Body: "data: {not json\n\n", SSE: true})
		r := drain(t, geminiClient(t, f, "gemini-4-carbon"), longReq())
		if r.Err == nil || !strings.Contains(r.Err.Error(), "malformed") {
			t.Errorf("err = %v, want a malformed-event error", r.Err)
		}
	})
	t.Run("stream/in-band error", func(t *testing.T) {
		f := newFake(t, sseReply(first), sseReply(`{"error":{"code":503,"message":"overloaded","status":"UNAVAILABLE"}}`))
		r := drain(t, geminiClient(t, f, "gemini-4-carbon"), longReq())
		var apiErr *llm.APIError
		if !errors.As(r.Err, &apiErr) {
			t.Errorf("err = %v, want an APIError", r.Err)
		}
	})
}

// A stream that ends (cleanly or abruptly) inside a later hop is an error,
// not a quiet end of the turn.
func TestLongDecodeStreamCutMidHop(t *testing.T) {
	first := sseReply(chunk("a", "CONTINUATION", "T1", nil))
	t.Run("clean EOF without finishReason", func(t *testing.T) {
		f := newFake(t, first, sseReply(chunk("b", "", "", nil)))
		r := drain(t, geminiClient(t, f, "gemini-4-carbon"), longReq())
		if !errors.Is(r.Err, io.ErrUnexpectedEOF) || len(r.Dones) != 0 {
			t.Errorf("err = %v dones %d, want ErrUnexpectedEOF and no Done", r.Err, len(r.Dones))
		}
	})
	t.Run("connection dropped mid-body", func(t *testing.T) {
		cut := reply{Body: sse(chunk("b", "", "", nil)), SSE: true, Cut: true}
		f := newFake(t, first, cut)
		r := drain(t, geminiClient(t, f, "gemini-4-carbon"), longReq())
		if r.Err == nil || len(r.Dones) != 0 {
			t.Errorf("err = %v dones %d, want an error and no Done", r.Err, len(r.Dones))
		}
		if r.Text != "ab" {
			t.Errorf("text %q: events before the cut are delivered", r.Text)
		}
	})
	t.Run("cut during the first hop", func(t *testing.T) {
		f := newFake(t, reply{Body: sse(chunk("a", "", "", nil)), SSE: true, Cut: true})
		r := drain(t, geminiClient(t, f, "gemini-4-carbon"), longReq())
		if r.Err == nil || f.hits() != 1 {
			t.Errorf("err = %v hits %d", r.Err, f.hits())
		}
	})
}

// A consumer that stops reading ends the turn: no further hop is requested.
func TestLongDecodeStreamConsumerBreaksEarly(t *testing.T) {
	endless := sseReply(chunk("a", "", "", nil), chunk("b", "CONTINUATION", "T1", nil))

	t.Run("during hop 1", func(t *testing.T) {
		f := newFake(t, endless)
		c := geminiClient(t, f, "gemini-4-carbon")
		for range c.Stream(t.Context(), longReq()) {
			break
		}
		if f.hits() != 1 {
			t.Errorf("%d requests after break, want 1", f.hits())
		}
	})
	t.Run("after hop 1's last event", func(t *testing.T) {
		f := newFake(t, endless)
		c := geminiClient(t, f, "gemini-4-carbon")
		n := 0
		for ev, err := range c.Stream(t.Context(), longReq()) {
			if err != nil {
				t.Fatal(err)
			}
			if ev.Type == llm.EventContent {
				n++
			}
			if n == 2 { // both hop-1 texts consumed, then stop
				break
			}
		}
		if f.hits() > 2 {
			t.Errorf("%d requests, want at most 2 (the break must stop the loop of hops)", f.hits())
		}
		time.Sleep(50 * time.Millisecond) // a runaway hop loop would keep hitting the server
		if h := f.hits(); h > 2 {
			t.Errorf("%d requests 50ms later: the hop loop kept running", h)
		}
	})
	t.Run("mid hop 2", func(t *testing.T) {
		f := newFake(t, endless)
		c := geminiClient(t, f, "gemini-4-carbon")
		n := 0
		for ev, err := range c.Stream(t.Context(), longReq()) {
			if err != nil {
				t.Fatal(err)
			}
			if ev.Type == llm.EventContent {
				n++
				if n == 3 { // first text of hop 2
					break
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
		if h := f.hits(); h != 2 {
			t.Errorf("%d requests, want exactly 2", h)
		}
	})
}

// Cancelling between hops stops the turn with the context's error.
func TestLongDecodeContextCancelledBetweenHops(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f := newFake(t)
	f.dynamic = func(n int, _ recorded) reply {
		if n == 1 {
			return reply{Do: cancel, Body: chunk("y", "CONTINUATION", "T2", nil)}
		}
		return jsonReply(chunk("x", "CONTINUATION", "T1", nil))
	}
	_, err := geminiClient(t, f, "gemini-4-carbon").Complete(ctx, longReq())
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if f.hits() > 3 {
		t.Errorf("%d requests after cancel, want at most 3", f.hits())
	}
}

// Missing, partial and negative usage across hops never corrupts the totals.
func TestLongDecodeUsageEdgeCases(t *testing.T) {
	cases := map[string]struct {
		hops        [2]string
		wantIn, out int
	}{
		"no hop reports": {
			[2]string{chunk("a", "CONTINUATION", "T1", nil), chunk("b", "STOP", "", nil)},
			llm.TokensNotReported, llm.TokensNotReported},
		"only the first": {
			[2]string{chunk("a", "CONTINUATION", "T1", &usage{In: 7, Out: 3}), chunk("b", "STOP", "", nil)},
			7, 3},
		"only the last": {
			[2]string{chunk("a", "CONTINUATION", "T1", nil), chunk("b", "STOP", "", &usage{In: 7, Out: 3})},
			7, 3},
		"negative everywhere": {
			[2]string{chunk("a", "CONTINUATION", "T1", &usage{In: -5, Out: -1, Think: -9, Cached: -2}),
				chunk("b", "STOP", "", &usage{In: -5, Out: -1, Think: -9, Cached: -2})},
			0, 0},
		"negative then positive": {
			[2]string{chunk("a", "CONTINUATION", "T1", &usage{In: -50, Out: -10}), chunk("b", "STOP", "", &usage{In: 7, Out: 3})},
			7, 3},
		"cached exceeds prompt": {
			[2]string{chunk("a", "CONTINUATION", "T1", &usage{In: 5, Out: 1, Cached: 99}), chunk("b", "STOP", "", &usage{In: 5, Out: 1, Cached: 99})},
			0, 2},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			res := runBoth(t, func(stream bool) *fakeAPI {
				return newFake(t, rp(stream, tc.hops[0]), rp(stream, tc.hops[1]))
			}, "gemini-4-carbon", nil)
			c := res["complete"]
			if c.O.Err != nil || c.O.In != tc.wantIn || c.O.Out != tc.out {
				t.Errorf("complete: %+v, want in %d out %d", c.O, tc.wantIn, tc.out)
			}
			// Same script as SSE, with the full Done event checked.
			f := newFake(t, sseReply(tc.hops[0]), sseReply(tc.hops[1]))
			r := drain(t, geminiClient(t, f, "gemini-4-carbon"), longReq())
			if r.Err != nil || len(r.Dones) != 1 {
				t.Fatalf("stream: err %v dones %d", r.Err, len(r.Dones))
			}
			if d := r.Dones[0]; d.InputTokens != tc.wantIn || d.OutputTokens != tc.out {
				t.Errorf("stream done in %d out %d, want %d/%d", d.InputTokens, d.OutputTokens, tc.wantIn, tc.out)
			}
			if r.Dones[0].InputTokens < llm.TokensNotReported || r.Dones[0].ThinkingTokens < 0 || r.Dones[0].CacheReadTokens < 0 {
				t.Errorf("negative leaked into %+v", r.Dones[0])
			}
		})
	}
}

// Sixteen long-decode turns share one client without mixing their hops (-race).
func TestLongDecodeConcurrent(t *testing.T) {
	f := newFake(t)
	var mu sync.Mutex
	var mixed []string
	f.dynamic = func(_ int, r recorded) reply {
		who := dig(r.Body, "contents")
		text := dig(map[string]any{"c": who.([]any)[0]}, "c", "parts")
		name := text.([]any)[0].(map[string]any)["text"].(string)
		token, _ := r.Body["continuationToken"].(string)
		stream := r.RawQuery == "alt=sse"
		mk := func(txt, fin, tok string) reply {
			if stream {
				return sseReply(chunk(txt, fin, tok, nil))
			}
			return jsonReply(chunk(txt, fin, tok, nil))
		}
		if token == "" {
			return mk(name+"-1 ", "CONTINUATION", "tok-"+name)
		}
		if token != "tok-"+name {
			mu.Lock()
			mixed = append(mixed, name+" got "+token)
			mu.Unlock()
		}
		return mk(name+"-2", "STOP", "")
	}
	c := geminiClient(t, f, "gemini-4-carbon")
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Go(func() {
			who := fmt.Sprintf("r%d", i)
			req := userReq(who)
			want := who + "-1 " + who + "-2"
			if i%2 == 0 {
				resp, err := c.Complete(t.Context(), req)
				if err != nil || resp.Content != want {
					t.Errorf("%s complete: %q %v", who, resp.Content, err)
				}
				return
			}
			r := drain(t, c, req)
			if r.Err != nil || r.Text != want {
				t.Errorf("%s stream: %q %v", who, r.Text, r.Err)
			}
		})
	}
	wg.Wait()
	if len(mixed) > 0 {
		t.Errorf("tokens crossed between turns: %v", mixed)
	}
	if f.hits() != 32 {
		t.Errorf("%d requests, want 32", f.hits())
	}
}

// A later hop with no candidate at all (an empty object, no promptFeedback):
// the turn must not silently lose what the earlier hops already produced.
// The text is kept and the turn ends as max_tokens (incomplete). Found here:
// mergeHop used to drop every earlier hop (fixed in rho-llm v0.20.0).
func TestLongDecodeEmptyLaterHopKeepsEarlierText(t *testing.T) {
	f := newFake(t, jsonReply(chunk("valuable text", "CONTINUATION", "T1", nil)), jsonReply(`{}`))
	resp, err := geminiClient(t, f, "gemini-4-carbon").Complete(t.Context(), longReq())
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "valuable text" || resp.StopReason != llm.StopMaxTokens {
		t.Errorf("content %q stop %q, want hop 1's text and max_tokens", resp.Content, resp.StopReason)
	}
}

// With default retries, a transient failure on a later hop restarts the whole
// turn from hop 1 (the call is retried as a unit): the second attempt is a
// fresh request with no token, and the final answer is still correct.
func TestLongDecodeRetryRestartsTheTurn(t *testing.T) {
	f := newFake(t,
		jsonReply(chunk("a", "CONTINUATION", "T1", nil)),
		statusReply(http.StatusServiceUnavailable, `{"error":{"code":503,"message":"busy","status":"UNAVAILABLE"}}`),
		jsonReply(chunk("a", "CONTINUATION", "T1b", nil)),
		jsonReply(chunk("b", "STOP", "", nil)),
	)
	c := geminiClient(t, f, "gemini-4-carbon", func(c *llm.Config) {
		c.DisableRetries = false
		c.RetryPolicy = &llm.RetryPolicy{BaseDelay: time.Millisecond, MaxDelay: time.Millisecond, Factor: 1}
	})
	resp, err := c.Complete(t.Context(), longReq())
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ab" || f.hits() != 4 {
		t.Errorf("content %q hits %d, want \"ab\" after 4 requests", resp.Content, f.hits())
	}
	if _, has := f.req(2).Body["continuationToken"]; has {
		t.Error("the retried attempt must start over without the old token")
	}
}
