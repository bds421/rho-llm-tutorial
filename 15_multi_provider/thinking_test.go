package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

// TestGeminiThinkingModelReturnsContent verifies that the Gemini adapter's
// maxOutputTokens padding produces non-empty content for thinking models.
// Without padding, a small MaxTokens budget is consumed entirely by the
// model's internal reasoning, returning empty Content and 0 output tokens.
//
// Requires GEMINI_API_KEY; skipped otherwise.
func TestGeminiThinkingModelReturnsContent(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	client, err := llm.NewClient(llm.Config{
		Provider:  "gemini",
		Model:     "gemini-2.5-flash",
		APIKey:    apiKey,
		MaxTokens: 100,
		Timeout:   30 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	resp, err := client.Complete(context.Background(), llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "What is 2+2? Answer with just the number."),
		},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if resp.Content == "" {
		t.Errorf("Content is empty — thinking may have consumed the entire output budget (out=%d)", resp.OutputTokens)
	}
	if resp.OutputTokens == 0 {
		t.Errorf("OutputTokens = 0 — Gemini reported no candidate tokens")
	}
	t.Logf("Content=%q, OutputTokens=%d, InputTokens=%d", resp.Content, resp.OutputTokens, resp.InputTokens)
}

// TestOllamaThinkingModelReturnsContent verifies that Ollama reasoning models
// produce non-empty Content when given sufficient token budget, and that the
// reasoning field is parsed into resp.Thinking.
//
// Requires a running Ollama instance with qwen3:4b; skipped otherwise.
func TestOllamaThinkingModelReturnsContent(t *testing.T) {
	// Quick check: is Ollama reachable?
	client, err := llm.NewClient(llm.Config{
		Provider:  "ollama",
		Model:     "qwen3:4b",
		MaxTokens: 2048,
		Timeout:   60 * time.Second,
	})
	if err != nil {
		t.Skip("Ollama not available: ", err)
	}
	defer client.Close()

	resp, err := client.Complete(context.Background(), llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "What is 2+2? Answer with just the number."),
		},
	})
	if err != nil {
		t.Skip("Ollama request failed (model may not be pulled): ", err)
	}

	if resp.Content == "" {
		t.Errorf("Content is empty — reasoning consumed all %d output tokens; increase MaxTokens", resp.OutputTokens)
	}
	if resp.Thinking == "" {
		t.Errorf("Thinking is empty — Qwen3 should produce reasoning content")
	}
	t.Logf("Content=%q, Thinking=%q (len=%d), OutputTokens=%d",
		resp.Content, truncate(resp.Thinking, 80), len(resp.Thinking), resp.OutputTokens)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
