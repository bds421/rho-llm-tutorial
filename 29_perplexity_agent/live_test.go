package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
)

func liveKey(t *testing.T, env string) string {
	t.Helper()
	if testing.Short() {
		t.Skip("live test: skipped under -short")
	}
	k := os.Getenv(env)
	if k == "" {
		t.Skipf("%s not set", env)
	}
	return k
}

func TestLivePerplexitySonar(t *testing.T) {
	key := liveKey(t, "PERPLEXITY_API_KEY")
	c, err := llm.NewClient(llm.Config{Provider: "perplexity", Model: "sonar", APIKey: key, MaxTokens: 100})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, err := c.Complete(ctx, userReq("Reply with the single word: ok"))
	if err != nil || resp.Content == "" {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
}

func TestLivePerplexityAgent(t *testing.T) {
	key := liveKey(t, "PERPLEXITY_API_KEY")
	c, err := llm.NewClient(llm.Config{Provider: "perplexity-agent", Model: "perplexity/sonar", APIKey: key, MaxTokens: 200})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req := userReq("Reply with the single word: ok")
	req.System = "Be terse."
	req.SamplingParams = map[string]any{"max_steps": 1}
	resp, err := c.Complete(ctx, req)
	if err != nil || resp.Content == "" {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
	if resp.InputTokens < 0 || resp.CacheReadTokens < 0 {
		t.Errorf("usage = %d/%d", resp.InputTokens, resp.CacheReadTokens)
	}
}

func TestLivePerplexityAgentStream(t *testing.T) {
	key := liveKey(t, "PERPLEXITY_API_KEY")
	c, err := llm.NewClient(llm.Config{Provider: "perplexity-agent", Model: "perplexity/sonar", APIKey: key, MaxTokens: 200})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	r := collect(c, userReq("Reply with the single word: ok"))
	if len(r.errs) != 0 || r.done == nil || r.text == "" {
		t.Fatalf("text=%q done=%v errs=%v", r.text, r.done, r.errs)
	}
}

// The shared Responses stream changes (call delivered by output_item.done or
// the terminal response) must not duplicate a plain OpenAI streamed call.
func TestLiveOpenAIStreamedToolCallExactlyOnce(t *testing.T) {
	key := liveKey(t, "OPENAI_API_KEY")
	c, err := llm.NewClient(llm.Config{Provider: "openai", Model: "gpt-6-luna", APIKey: key, MaxTokens: 600})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	req := userReq("What is the weather in Paris? Use the tool.")
	req.Tools = []llm.Tool{{
		Name:        "get_weather",
		Description: "Get the current weather for a city",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"city": map[string]any{"type": "string"}},
			"required":   []string{"city"},
		},
	}}
	req.ToolChoice = &llm.ToolChoice{Mode: llm.ToolChoiceRequired}
	r := collect(c, req)
	if len(r.errs) != 0 {
		t.Fatalf("errs: %v", r.errs)
	}
	if len(r.calls) != 1 || r.calls[0].Name != "get_weather" {
		t.Fatalf("calls = %+v, want exactly one get_weather", r.calls)
	}
	if m, _ := r.calls[0].Input.(map[string]any); m["city"] == nil {
		t.Errorf("input = %v", r.calls[0].Input)
	}
}
