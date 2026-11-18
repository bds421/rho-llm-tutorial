package main

import (
	"encoding/json"
	"testing"

	"github.com/bds421/rho-llm"
)

// Gemini finish reasons -> the neutral set; RawStopReason keeps the original.
func TestFinishReasonMapping(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"STOP", llm.StopEndTurn},
		{"MAX_TOKENS", llm.StopMaxTokens},
		{"CONTINUATION", llm.StopMaxTokens}, // no token: truncated, not failed
		{"SAFETY", llm.StopRefusal},
		{"PROHIBITED_CONTENT", llm.StopRefusal},
		{"RECITATION", llm.StopRefusal},
		{"BLOCKLIST", llm.StopRefusal},
		{"SPII", llm.StopRefusal},
		{"ESCALATION", llm.StopRefusal},              // v0.20.0
		{"PUP_LIMITED_DISABLED", llm.StopRefusal},    // v0.20.0
		{"MISSING_THOUGHT_SIGNATURE", llm.StopError}, // v0.20.0
		{"MALFORMED_RESPONSE", llm.StopError},        // v0.20.0
		{"MALFORMED_FUNCTION_CALL", llm.StopError},
		{"OTHER", llm.StopError},
		{"LANGUAGE", llm.StopError},
		{"FINISH_REASON_UNSPECIFIED", llm.StopError},
		{"A_REASON_GOOGLE_INVENTS_NEXT_YEAR", llm.StopError}, // unknown must not look like a normal stop
		{"stop", llm.StopError},                              // case matters: wire values are upper case
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			body := chunk("t", tc.raw, "", nil)

			resp, err := geminiClient(t, newFake(t, jsonReply(body)), "gemini-4-carbon").Complete(t.Context(), userReq("hi"))
			if err != nil {
				t.Fatal(err)
			}
			if resp.StopReason != tc.want || resp.RawStopReason != tc.raw {
				t.Errorf("complete: stop %q raw %q, want %q / %q", resp.StopReason, resp.RawStopReason, tc.want, tc.raw)
			}

			r := drain(t, geminiClient(t, newFake(t, sseReply(body)), "gemini-4-carbon"), userReq("hi"))
			if r.Err != nil || len(r.Dones) != 1 {
				t.Fatalf("stream: err %v dones %d", r.Err, len(r.Dones))
			}
			if d := r.Dones[0]; d.StopReason != tc.want || d.RawStopReason != tc.raw {
				t.Errorf("stream: stop %q raw %q, want %q / %q", d.StopReason, d.RawStopReason, tc.want, tc.raw)
			}
		})
	}
}

// Only a normal STOP becomes tool_use when a call is present; a refusal, an
// error or a truncation with a call in it keeps its own reason.
func TestToolCallDoesNotMaskFailureReasons(t *testing.T) {
	cases := map[string]string{
		"STOP":                      llm.StopToolUse,
		"MAX_TOKENS":                llm.StopMaxTokens,
		"ESCALATION":                llm.StopRefusal,
		"MALFORMED_RESPONSE":        llm.StopError,
		"MISSING_THOUGHT_SIGNATURE": llm.StopError,
	}
	for raw, want := range cases {
		t.Run(raw, func(t *testing.T) {
			body := callChunk("lookup", map[string]any{"q": 1}, raw, "")
			resp, err := geminiClient(t, newFake(t, jsonReply(body)), "gemini-4-carbon").Complete(t.Context(), userReq("hi"))
			if err != nil || resp.StopReason != want {
				t.Errorf("complete: %v stop %q, want %q", err, resp.StopReason, want)
			}
			r := drain(t, geminiClient(t, newFake(t, sseReply(body)), "gemini-4-carbon"), userReq("hi"))
			if r.Err != nil || len(r.Dones) != 1 || r.Dones[0].StopReason != want {
				t.Errorf("stream: err %v dones %+v, want %q", r.Err, r.Dones, want)
			}
		})
	}
}

// A blocked prompt (no candidates, a blockReason) is a refusal, never an
// empty successful answer.
func TestBlockedPromptIsARefusal(t *testing.T) {
	body := `{"promptFeedback":{"blockReason":"PROHIBITED_CONTENT"},"usageMetadata":{"promptTokenCount":4}}`
	resp, err := geminiClient(t, newFake(t, jsonReply(body)), "gemini-4-carbon").Complete(t.Context(), userReq("hi"))
	if err != nil || resp.StopReason != llm.StopRefusal || resp.RawStopReason != "PROHIBITED_CONTENT" {
		t.Errorf("complete: %v %+v", err, resp)
	}
	r := drain(t, geminiClient(t, newFake(t, sseReply(body)), "gemini-4-carbon"), userReq("hi"))
	if r.Err != nil || len(r.Dones) != 1 || r.Dones[0].StopReason != llm.StopRefusal {
		t.Errorf("stream: err %v dones %+v", r.Err, r.Dones)
	}
}

// Hostile bodies: empty candidates, empty parts, a finish reason with no
// content, JSON null where objects belong. None may panic.
func TestHostileResponseShapesDoNotPanic(t *testing.T) {
	bodies := []string{
		`{}`,
		`{"candidates":[]}`,
		`{"candidates":[{}]}`,
		`{"candidates":[{"content":{"parts":[]},"finishReason":"STOP"}]}`,
		`{"candidates":[{"content":null,"finishReason":"STOP"}]}`,
		`{"candidates":[null]}`,
		`{"candidates":[{"content":{"parts":[{}]},"finishReason":"STOP"}]}`,
		`{"candidates":[{"content":{"parts":[{"functionCall":null}]},"finishReason":"STOP"}]}`,
		`{"candidates":[{"content":{"parts":[{"text":"x"}]},"finishReason":"STOP"}],"usageMetadata":null}`,
		`null`,
		`[]`,
	}
	for i, b := range bodies {
		func() {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("body %d %s panicked in Complete: %v", i, b, p)
				}
			}()
			_, _ = geminiClient(t, newFake(t, jsonReply(b)), "gemini-4-carbon").Complete(t.Context(), userReq("hi"))
		}()
		func() {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("body %d %s panicked in Stream: %v", i, b, p)
				}
			}()
			drain(t, geminiClient(t, newFake(t, sseReply(b)), "gemini-4-carbon"), userReq("hi"))
		}()
	}
	var sink map[string]any
	_ = json.Unmarshal([]byte(`{}`), &sink)
}
