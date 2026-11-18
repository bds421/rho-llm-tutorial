package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
)

// liveClient skips unless -short is off and GEMINI_API_KEY is set. Run:
//
//	set -a; . ../.env; set +a; go test -count=1 -run Live -v ./...
func liveClient(t *testing.T, provider, model string) (llm.Client, context.Context) {
	t.Helper()
	if testing.Short() {
		t.Skip("live test: skipped under -short")
	}
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("live test: GEMINI_API_KEY not set")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	c, err := llm.NewClient(llm.Config{Provider: provider, Model: model, APIKey: key, DisableRetries: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, ctx
}

func okReq(level llm.ThinkingLevel) llm.Request {
	return llm.Request{
		Messages:      []llm.Message{llm.NewTextMessage(llm.RoleUser, "Reply with the single word ok.")},
		MaxTokens:     200,
		ThinkingLevel: level,
	}
}

// Levels that Google answers with a 400 (minimal on 3.8 Flash and Gemini 4;
// xhigh on everything but Hydrogen) must come back as a normal answer.
func TestLiveThinkingLevels(t *testing.T) {
	cases := []struct {
		model string
		level llm.ThinkingLevel
	}{
		{"gemini-3.8-flash", llm.ThinkingMinimal},
		{"gemini-3.8-flash", llm.ThinkingXHigh},
		{"gemini-4-hydrogen", llm.ThinkingXHigh},
		{"gemini-4-hydrogen", llm.ThinkingMinimal},
	}
	for _, tc := range cases {
		t.Run(tc.model+"/"+string(tc.level), func(t *testing.T) {
			c, ctx := liveClient(t, "gemini", tc.model)
			resp, err := c.Complete(ctx, okReq(tc.level))
			if err != nil {
				t.Fatalf("%s at %s: %v", tc.model, tc.level, err)
			}
			if resp.StopReason != llm.StopEndTurn || strings.TrimSpace(resp.Content) == "" {
				t.Errorf("stop %q raw %q content %q", resp.StopReason, resp.RawStopReason, resp.Content)
			}
			if resp.OutputTokens <= 0 || resp.InputTokens < 0 {
				t.Errorf("usage in=%d out=%d", resp.InputTokens, resp.OutputTokens)
			}
		})
	}
}

// Gemini 4 streams through the same long-decode loop; one hop is the common case.
func TestLiveGemini4HydrogenStream(t *testing.T) {
	c, ctx := liveClient(t, "gemini", "gemini-4-hydrogen")
	r := drain(t, c, okReq(llm.ThinkingXHigh))
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if len(r.Dones) != 1 || r.Dones[0].StopReason != llm.StopEndTurn || strings.TrimSpace(r.Text) == "" {
		t.Errorf("text %q dones %+v", r.Text, r.Dones)
	}
	_ = ctx
}

func TestLiveInteractionsCompleteAndStream(t *testing.T) {
	c, ctx := liveClient(t, "gemini-interactions", "gemini-3.8-flash")
	if c.Provider() != "gemini-interactions" {
		t.Errorf("Provider() = %q", c.Provider())
	}
	req := llm.Request{
		Messages:      []llm.Message{llm.NewTextMessage(llm.RoleUser, "What is 17*23? Answer with just the number.")},
		MaxTokens:     200,
		ThinkingLevel: llm.ThinkingLow,
	}
	resp, err := c.Complete(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Content, "391") || resp.StopReason != llm.StopEndTurn || resp.RawStopReason != "completed" {
		t.Errorf("complete: %q stop %q raw %q", resp.Content, resp.StopReason, resp.RawStopReason)
	}
	if resp.OutputTokens <= 0 {
		t.Errorf("output tokens %d", resp.OutputTokens)
	}

	var text string
	var done llm.StreamEvent
	for ev, err := range c.Stream(ctx, req) {
		if err != nil {
			t.Fatal(err)
		}
		switch ev.Type {
		case llm.EventContent:
			text += ev.Text
		case llm.EventDone:
			done = ev
		}
	}
	if !strings.Contains(text, "391") || done.StopReason != llm.StopEndTurn {
		t.Errorf("stream: %q done %+v", text, done)
	}
}

// Interactions tool-free, thinking at the model-specific levels: a 400 for
// "minimal" or "xhigh" on 3.8 Flash would surface here.
func TestLiveInteractionsThinkingLevels(t *testing.T) {
	for _, level := range []llm.ThinkingLevel{llm.ThinkingMinimal, llm.ThinkingXHigh} {
		t.Run(string(level), func(t *testing.T) {
			c, ctx := liveClient(t, "gemini-interactions", "gemini-3.8-flash")
			resp, err := c.Complete(ctx, okReq(level))
			if err != nil || strings.TrimSpace(resp.Content) == "" {
				t.Fatalf("level %s: %v %+v", level, err, resp)
			}
		})
	}
}

// Probe, not an assertion: SamplingParams["temperature"] is refused on the
// generateContent provider because it lands at the top level of the body. The
// interactions adapter merges SamplingParams the same way; this logs what
// Google does with it there.
func TestLiveInteractionsTopLevelTemperatureProbe(t *testing.T) {
	c, ctx := liveClient(t, "gemini-interactions", "gemini-3.8-flash")
	req := okReq(llm.ThinkingNone)
	req.SamplingParams = map[string]any{"temperature": 0.5}
	if _, err := c.Complete(ctx, req); err == nil {
		t.Fatal("SamplingParams[temperature] accepted on interactions (Google answers 400 Unknown parameter)")
	} else {
		t.Logf("refused: %v", err)
	}
}
