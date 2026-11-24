package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/bds421/rho-llm"
)

// interaction builds a completed-interaction JSON body (the shape of a live
// response, shortened).
func interaction(status, text string, u *[4]int) string {
	m := map[string]any{
		"object": "interaction", "status": status, "model": "gemini-3.8-flash",
		"steps": []any{
			map[string]any{"type": "thought", "signature": "SIG1", "summary": []any{map[string]any{"type": "text", "text": "pondering "}}},
			map[string]any{"type": "model_output", "content": []any{map[string]any{"type": "text", "text": text}}},
		},
	}
	if u != nil {
		m["usage"] = map[string]any{
			"total_input_tokens": u[0], "total_output_tokens": u[1],
			"total_thought_tokens": u[2], "total_cached_tokens": u[3],
		}
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func ev(kind string, rest string) string {
	if rest != "" {
		rest = "," + rest
	}
	return fmt.Sprintf(`{"event_type":%q%s}`, kind, rest)
}

// interactionSSE is a complete streamed turn: thought summary, signature, two
// text deltas, usage on interaction.completed, then [DONE].
func interactionSSE(status string, usageJSON string) string {
	return sse(
		ev("interaction.created", `"interaction":{"model":"gemini-3.8-flash"}`),
		ev("step.start", `"index":0,"step":{"type":"thought"}`),
		ev("step.delta", `"index":0,"delta":{"type":"thought_summary","content":{"type":"text","text":"hmm "}}`),
		ev("step.delta", `"index":0,"delta":{"type":"thought_signature","signature":"SIGS"}`),
		ev("step.stop", `"index":0`),
		ev("step.start", `"index":1,"step":{"type":"model_output"}`),
		ev("step.delta", `"index":1,"delta":{"type":"text","text":"Hel"}`),
		ev("step.delta", `"index":1,"delta":{"type":"text","text":"lo"}`),
		ev("step.stop", `"index":1`),
		ev("interaction.completed", fmt.Sprintf(`"interaction":{"status":%q%s}`, status, usageJSON)),
		`[DONE]`,
	)
}

const usageJSON = `,"usage":{"total_input_tokens":12,"total_cached_tokens":2,"total_output_tokens":5,"total_thought_tokens":7}`

// Presets: the opt-in name selects the interactions protocol, "gemini" and
// "google" stay on generateContent, and the two never cross paths.
func TestPresetsSelectProtocolAndPath(t *testing.T) {
	for name, want := range map[string]string{
		"gemini": "gemini", "google": "gemini", "gemini-interactions": "gemini_interactions",
	} {
		if got := llm.ResolveProtocol(llm.Config{Provider: name}); got != want {
			t.Errorf("ResolveProtocol(%s) = %q, want %q", name, got, want)
		}
	}
	if got := llm.ResolveBaseURL(llm.Config{Provider: "gemini-interactions"}); got != "https://generativelanguage.googleapis.com/v1beta" {
		t.Errorf("interactions base %q (an API root: the adapter appends /interactions)", got)
	}

	fi := newFake(t, jsonReply(interaction("completed", "hi", nil)))
	if _, err := interactionsClient(t, fi, "gemini-3.8-flash").Complete(t.Context(), userReq("hi")); err != nil {
		t.Fatal(err)
	}
	if p := fi.req(0).Path; p != "/v1beta/interactions" || fi.req(0).Method != "POST" || !fi.req(0).HasKey {
		t.Errorf("interactions request: %s %q key=%v", fi.req(0).Method, p, fi.req(0).HasKey)
	}

	fg := newFake(t, jsonReply(okBody))
	if _, err := geminiClient(t, fg, "gemini-3.8-flash").Complete(t.Context(), userReq("hi")); err != nil {
		t.Fatal(err)
	}
	if p := fg.req(0).Path; strings.Contains(p, "interactions") || !strings.HasSuffix(p, ":generateContent") {
		t.Errorf("the gemini provider used %q", p)
	}
	if got := interactionsClient(t, fi, "gemini-3.8-flash").Provider(); got != "gemini-interactions" {
		t.Errorf("Provider() = %q", got)
	}
}

func TestInteractionsCompleteEndToEnd(t *testing.T) {
	f := newFake(t, jsonReply(interaction("completed", "391", &[4]int{14, 270, 809, 4})))
	c := interactionsClient(t, f, "gemini-3.8-flash")
	req := userReq("17*23?")
	req.System = "Be brief."
	req.ThinkingLevel = llm.ThinkingHigh
	resp, err := c.Complete(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	b := f.req(0).Body
	if b["store"] != false {
		t.Errorf("store = %v: the default must be stateless (the API's own default is true)", b["store"])
	}
	if b["model"] != "gemini-3.8-flash" || b["system_instruction"] != "Be brief." {
		t.Errorf("body %v", b)
	}
	if dig(b, "generation_config", "thinking_level") != "high" || dig(b, "generation_config", "thinking_summaries") != "auto" {
		t.Errorf("generation_config %v", b["generation_config"])
	}
	if _, has := b["contents"]; has {
		t.Error("interactions bodies carry input steps, not contents")
	}
	if resp.Content != "391" || resp.Thinking != "pondering " || resp.ThinkingSignature != "SIG1" {
		t.Errorf("resp %+v", resp)
	}
	// total_input_tokens includes the cached share: 14 - 4 uncached, 4 cache reads.
	if resp.InputTokens != 10 || resp.CacheReadTokens != 4 || resp.OutputTokens != 270 || resp.ThinkingTokens != 809 {
		t.Errorf("usage in=%d cache=%d out=%d think=%d", resp.InputTokens, resp.CacheReadTokens, resp.OutputTokens, resp.ThinkingTokens)
	}
	if resp.StopReason != llm.StopEndTurn || resp.RawStopReason != "completed" {
		t.Errorf("stop %q raw %q", resp.StopReason, resp.RawStopReason)
	}
}

// store: false by default, true only on an explicit boolean opt-in; any other
// type is refused before the wire, and "store" never leaks in twice.
func TestInteractionsStoreOptIn(t *testing.T) {
	for name, tc := range map[string]struct {
		params  map[string]any
		want    any
		refused bool
	}{
		"default":   {nil, false, false},
		"empty map": {map[string]any{}, false, false},
		"true":      {map[string]any{"store": true}, true, false},
		"false":     {map[string]any{"store": false}, false, false},
		"string":    {map[string]any{"store": "true"}, nil, true},
		"number":    {map[string]any{"store": 1}, nil, true},
		"nil":       {map[string]any{"store": nil}, nil, true},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, jsonReply(interaction("completed", "x", nil)))
			c := interactionsClient(t, f, "gemini-3.8-flash")
			req := userReq("hi")
			req.SamplingParams = tc.params
			_, err := c.Complete(t.Context(), req)
			if tc.refused {
				if err == nil || f.hits() != 0 {
					t.Errorf("err = %v hits %d, want a refusal before the wire", err, f.hits())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if f.req(0).Body["store"] != tc.want {
				t.Errorf("store = %v, want %v", f.req(0).Body["store"], tc.want)
			}
		})
	}
	// The opt-in rides on a stream request too.
	f := newFake(t, sseReply(ev("interaction.completed", `"interaction":{"status":"completed"}`)))
	req := userReq("hi")
	req.SamplingParams = map[string]any{"store": true}
	if r := drain(t, interactionsClient(t, f, "gemini-3.8-flash"), req); r.Err != nil {
		t.Fatal(r.Err)
	}
	if f.req(0).Body["store"] != true || f.req(0).Body["stream"] != true {
		t.Errorf("stream body %v", f.req(0).Body)
	}
}

// Features the adapter does not carry are refused before the request.
func TestInteractionsRefusedFeatures(t *testing.T) {
	cases := map[string]func(*llm.Request){
		"CachedContent":  func(r *llm.Request) { r.CachedContent = "cachedContents/x" },
		"ThinkingBudget": func(r *llm.Request) { r.ThinkingBudget = 2048 },
		"Effort":         func(r *llm.Request) { r.Effort = "high" },
		"structured output": func(r *llm.Request) {
			r.ResponseFormat = &llm.ResponseFormat{Type: llm.ResponseFormatJSONSchema, Schema: map[string]any{"type": "object"}}
		},
		"tool result with no matching call": func(r *llm.Request) {
			r.Messages = append(r.Messages, llm.NewToolResultMessage("ghost", "x", false))
		},
		"empty conversation": func(r *llm.Request) { r.Messages = nil },
		"top-level key already set by the adapter": func(r *llm.Request) {
			r.SamplingParams = map[string]any{"model": "gemini-4-carbon"}
		},
		"reserved key": func(r *llm.Request) { r.SamplingParams = map[string]any{"input": []any{}} },
		"empty key":    func(r *llm.Request) { r.SamplingParams = map[string]any{"": 1} },
	}
	for name, mod := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, jsonReply(interaction("completed", "x", nil)))
			c := interactionsClient(t, f, "gemini-3.8-flash")
			req := userReq("hi")
			mod(&req)
			if _, err := c.Complete(t.Context(), req); err == nil {
				t.Error("Complete: want a refusal")
			}
			if r := drain(t, c, req); r.Err == nil {
				t.Error("Stream: want a refusal")
			}
			if f.hits() != 0 {
				t.Errorf("%d HTTP requests, want 0", f.hits())
			}
		})
	}
}

// The same per-model thinking rules as generateContent, in the interactions
// field names.
func TestInteractionsThinkingLevelPerModel(t *testing.T) {
	cases := []struct {
		model string
		level llm.ThinkingLevel
		want  string
	}{
		{"gemini-3.8-flash", llm.ThinkingMinimal, "low"},
		{"gemini-3.8-flash", llm.ThinkingXHigh, "high"},
		{"gemini-3.8-flash", llm.ThinkingMax, "high"},
		{"gemini-4-hydrogen", llm.ThinkingXHigh, "xhigh"},
		{"gemini-4-hydrogen", llm.ThinkingMinimal, "low"},
		{"gemini-4-neon", llm.ThinkingXHigh, "xhigh"},
		{"gemini-4-carbon", llm.ThinkingXHigh, "high"},
		{"gemini-4-carbon", llm.ThinkingMedium, "medium"},
	}
	for _, tc := range cases {
		t.Run(tc.model+"/"+string(tc.level), func(t *testing.T) {
			f := newFake(t, jsonReply(interaction("completed", "x", nil)))
			c := interactionsClient(t, f, tc.model)
			req := userReq("hi")
			req.ThinkingLevel = tc.level
			if _, err := c.Complete(t.Context(), req); err != nil {
				t.Fatal(err)
			}
			if got := dig(f.req(0).Body, "generation_config", "thinking_level"); got != tc.want {
				t.Errorf("thinking_level = %v, want %q", got, tc.want)
			}
		})
	}
}

func TestInteractionsStreamEndToEnd(t *testing.T) {
	f := newFake(t, sseReply2(interactionSSE("completed", usageJSON)))
	r := drain(t, interactionsClient(t, f, "gemini-3.8-flash"), userReq("hi"))
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.Text != "Hello" || r.Thinking != "hmm " {
		t.Errorf("text %q thinking %q", r.Text, r.Thinking)
	}
	if len(r.Dones) != 1 {
		t.Fatalf("%d Done events", len(r.Dones))
	}
	d := r.Dones[0]
	if d.StopReason != llm.StopEndTurn || d.RawStopReason != "completed" || d.ThinkingSignature != "SIGS" ||
		d.InputTokens != 10 || d.CacheReadTokens != 2 || d.OutputTokens != 5 || d.ThinkingTokens != 7 {
		t.Errorf("done %+v", d)
	}
	if f.req(0).Body["stream"] != true || f.req(0).Body["store"] != false {
		t.Errorf("body %v", f.req(0).Body)
	}
	acc := &llm.StreamAccumulator{}
	for _, e := range r.Events {
		acc.Add(e)
	}
	if got := acc.Response(); got.Content != "Hello" {
		t.Errorf("accumulated %q", got.Content)
	}
}

// sseReply2 serves an already-framed SSE body.
func sseReply2(body string) reply { return reply{Body: body, SSE: true} }

// Hostile streams: no completion event, a cut connection, garbage lines, an
// in-band error, an early break.
func TestInteractionsStreamHostile(t *testing.T) {
	t.Run("[DONE] without interaction.completed", func(t *testing.T) {
		f := newFake(t, sseReply2(sse(ev("step.delta", `"index":0,"delta":{"type":"text","text":"x"}`), `[DONE]`)))
		r := drain(t, interactionsClient(t, f, "gemini-3.8-flash"), userReq("hi"))
		if !errors.Is(r.Err, io.ErrUnexpectedEOF) || len(r.Dones) != 0 {
			t.Errorf("err %v dones %d, want ErrUnexpectedEOF", r.Err, len(r.Dones))
		}
	})
	t.Run("empty stream", func(t *testing.T) {
		f := newFake(t, sseReply2(""))
		if r := drain(t, interactionsClient(t, f, "gemini-3.8-flash"), userReq("hi")); !errors.Is(r.Err, io.ErrUnexpectedEOF) {
			t.Errorf("err %v", r.Err)
		}
	})
	t.Run("connection dropped", func(t *testing.T) {
		f := newFake(t, reply{Body: sse(ev("step.delta", `"index":0,"delta":{"type":"text","text":"x"}`)), SSE: true, Cut: true})
		r := drain(t, interactionsClient(t, f, "gemini-3.8-flash"), userReq("hi"))
		if r.Err == nil || len(r.Dones) != 0 || r.Text != "x" {
			t.Errorf("err %v dones %d text %q", r.Err, len(r.Dones), r.Text)
		}
	})
	t.Run("malformed event", func(t *testing.T) {
		f := newFake(t, sseReply2("data: {oops\n\n"))
		r := drain(t, interactionsClient(t, f, "gemini-3.8-flash"), userReq("hi"))
		if r.Err == nil || !strings.Contains(r.Err.Error(), "malformed") {
			t.Errorf("err %v", r.Err)
		}
	})
	t.Run("in-band error", func(t *testing.T) {
		f := newFake(t, sseReply2(sse(`{"error":{"message":"quota","code":"resource_exhausted"}}`)))
		r := drain(t, interactionsClient(t, f, "gemini-3.8-flash"), userReq("hi"))
		var apiErr *llm.APIError
		if !errors.As(r.Err, &apiErr) || apiErr.StatusCode != 429 {
			t.Errorf("err %v, want an APIError 429", r.Err)
		}
	})
	t.Run("early break", func(t *testing.T) {
		f := newFake(t, sseReply2(interactionSSE("completed", usageJSON)))
		c := interactionsClient(t, f, "gemini-3.8-flash")
		for range c.Stream(t.Context(), userReq("hi")) {
			break
		}
		if f.hits() != 1 {
			t.Errorf("%d requests", f.hits())
		}
	})
}

// Interaction status -> stop reason; RawStopReason keeps the status.
func TestInteractionsStatusMapping(t *testing.T) {
	for status, want := range map[string]string{
		"completed":       llm.StopEndTurn,
		"incomplete":      llm.StopMaxTokens,
		"failed":          llm.StopError,
		"cancelled":       llm.StopError,
		"requires_action": llm.StopError, // no function call present
		"in_progress":     llm.StopError,
		"brand_new":       llm.StopError,
		"":                "",
	} {
		t.Run("status="+status, func(t *testing.T) {
			f := newFake(t, jsonReply(interaction(status, "x", nil)))
			resp, err := interactionsClient(t, f, "gemini-3.8-flash").Complete(t.Context(), userReq("hi"))
			if err != nil || resp.StopReason != want || resp.RawStopReason != status {
				t.Errorf("%v: stop %q raw %q, want %q", err, resp.StopReason, resp.RawStopReason, want)
			}
			r := drain(t, interactionsClient(t, newFake(t, sseReply2(interactionSSE(status, ""))), "gemini-3.8-flash"), userReq("hi"))
			if status == "failed" || status == "cancelled" {
				return // those arrive as status updates in a stream
			}
			if r.Err != nil || len(r.Dones) != 1 || r.Dones[0].StopReason != want {
				t.Errorf("stream: err %v dones %+v, want %q", r.Err, r.Dones, want)
			}
		})
	}
}

// A function call: requires_action + a function_call step -> tool_use.
func TestInteractionsToolCall(t *testing.T) {
	body := `{"status":"requires_action","steps":[{"type":"thought","signature":"S"},{"type":"function_call","id":"c1","name":"lookup","arguments":{"q":"x"}}],"usage":{"total_input_tokens":3,"total_output_tokens":2}}`
	f := newFake(t, jsonReply(body))
	req := userReq("hi")
	req.Tools = []llm.Tool{{Name: "lookup", Description: "d", InputSchema: map[string]any{"type": "object"}}}
	resp, err := interactionsClient(t, f, "gemini-3.8-flash").Complete(t.Context(), req)
	if err != nil || resp.StopReason != llm.StopToolUse || len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "c1" {
		t.Errorf("%v %+v", err, resp)
	}
	if tools, _ := f.req(0).Body["tools"].([]any); len(tools) != 1 {
		t.Errorf("tools %v", f.req(0).Body["tools"])
	}
}

// Usage: missing -> TokensNotReported; negative and oversize cached clamp.
func TestInteractionsUsageEdgeCases(t *testing.T) {
	for name, tc := range map[string]struct {
		u               *[4]int
		in, out, cached int
	}{
		"missing":         {nil, llm.TokensNotReported, llm.TokensNotReported, 0},
		"negative":        {&[4]int{-5, -3, -2, -1}, 0, 0, 0},
		"cached > prompt": {&[4]int{5, 1, 0, 99}, 0, 1, 5},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, jsonReply(interaction("completed", "x", tc.u)))
			resp, err := interactionsClient(t, f, "gemini-3.8-flash").Complete(t.Context(), userReq("hi"))
			if err != nil {
				t.Fatal(err)
			}
			if resp.InputTokens != tc.in || resp.OutputTokens != tc.out || resp.CacheReadTokens != tc.cached || resp.ThinkingTokens < 0 {
				t.Errorf("usage in=%d out=%d cache=%d think=%d", resp.InputTokens, resp.OutputTokens, resp.CacheReadTokens, resp.ThinkingTokens)
			}
		})
	}
}

// HTTP and in-band errors surface as *APIError and never carry the key.
func TestInteractionsErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		r      reply
		status int
	}{
		"http 400":         {statusReply(400, `{"error":{"message":"bad","code":"invalid_request"}}`), 400},
		"http 404":         {statusReply(404, `{"error":{"message":"no such model","code":"not_found"}}`), 404},
		"http 429":         {statusReply(429, `{"error":{"message":"slow down"}}`), 429},
		"http 500 html":    {statusReply(500, `<html>oops</html>`), 500},
		"in-band in a 200": {jsonReply(`{"error":{"message":"nope","code":"invalid_request"}}`), 400},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, tc.r)
			_, err := interactionsClient(t, f, "gemini-3.8-flash").Complete(t.Context(), userReq("hi"))
			var apiErr *llm.APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != tc.status {
				t.Fatalf("err = %v, want an APIError %d", err, tc.status)
			}
			if strings.Contains(err.Error(), fakeKey) {
				t.Error("leaked the key")
			}
		})
	}
	for name, body := range map[string]string{"garbage": `{{{`, "empty": ``, "array": `[]`} {
		t.Run("200 "+name, func(t *testing.T) {
			f := newFake(t, jsonReply(body))
			if _, err := interactionsClient(t, f, "gemini-3.8-flash").Complete(t.Context(), userReq("hi")); err == nil {
				t.Error("want a decode error")
			}
		})
	}
}

// One client, sixteen concurrent turns, half streaming (-race).
func TestInteractionsConcurrent(t *testing.T) {
	f := newFake(t)
	f.dynamic = func(_ int, r recorded) reply {
		if r.Body["stream"] == true {
			return sseReply2(interactionSSE("completed", usageJSON))
		}
		return jsonReply(interaction("completed", "Hello", &[4]int{1, 1, 1, 0}))
	}
	c := interactionsClient(t, f, "gemini-3.8-flash")
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Go(func() {
			if i%2 == 0 {
				if resp, err := c.Complete(t.Context(), userReq("hi")); err != nil || resp.Content != "Hello" {
					t.Errorf("complete: %v %v", resp, err)
				}
				return
			}
			if r := drain(t, c, userReq("hi")); r.Err != nil || r.Text != "Hello" {
				t.Errorf("stream: %v %q", r.Err, r.Text)
			}
		})
	}
	wg.Wait()
	if f.hits() != 16 {
		t.Errorf("%d requests", f.hits())
	}
}

// Top-level temperature/top_p/top_k in SamplingParams are refused before the
// wire on interactions too: Google answers 400 "Unknown parameter
// 'temperature'" (live, see TestLiveInteractionsTopLevelTemperatureProbe).
// Found here, fixed in rho-llm v0.20.0.
func TestInteractionsRefusesTopLevelSamplingKeys(t *testing.T) {
	f := newFake(t, jsonReply(interaction("completed", "x", nil)))
	req := userReq("hi")
	req.SamplingParams = map[string]any{"temperature": 0.5}
	_, err := interactionsClient(t, f, "gemini-3.8-flash").Complete(t.Context(), req)
	if err == nil || f.hits() != 0 {
		t.Errorf("err = %v, hits %d: want a refusal before the request (Google answers 400)", err, f.hits())
	}
}
