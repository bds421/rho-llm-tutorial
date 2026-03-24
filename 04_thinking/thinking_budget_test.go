package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

// TestThinkingBudgetTokensDefaults verifies the pure helper function
// that resolves ThinkingLevel → token count. No API call needed.
func TestThinkingBudgetTokensDefaults(t *testing.T) {
	for _, tc := range []struct {
		level  llm.ThinkingLevel
		custom int
	}{
		{llm.ThinkingNone, 0},
		{llm.ThinkingMinimal, 0},
		{llm.ThinkingLow, 0},
		{llm.ThinkingMedium, 0},
		{llm.ThinkingHigh, 0},
		{llm.ThinkingXHigh, 0},
		{llm.ThinkingHigh, 5000}, // custom overrides level default
	} {
		tokens := llm.ThinkingBudgetTokens(tc.level, tc.custom)
		t.Logf("%-10s custom=%5d → %d tokens", tc.level, tc.custom, tokens)
		if tc.level != llm.ThinkingNone && tokens == 0 {
			t.Errorf("expected non-zero budget for level %q", tc.level)
		}
		if tc.custom > 0 && tokens != tc.custom {
			t.Errorf("custom=%d but got %d", tc.custom, tokens)
		}
	}
}

// TestThinkingBudgetAnthropic sends a per-request ThinkingBudget to Anthropic.
// Requires ANTHROPIC_API_KEY; skipped otherwise.
func TestThinkingBudgetAnthropic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in -short mode")
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	client, err := llm.NewClient(llm.Config{
		Provider:  "anthropic",
		Model:     "claude-haiku-4-5-20251001",
		APIKey:    apiKey,
		MaxTokens: 4096,
		Timeout:   60 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	resp, err := client.Complete(context.Background(), llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "What is 7 * 8? Answer with just the number."),
		},
		ThinkingLevel:  llm.ThinkingLow,
		ThinkingBudget: 1024,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if resp.Content == "" {
		t.Error("Content is empty")
	}
	if resp.Thinking == "" {
		t.Error("Thinking is empty — expected reasoning output with ThinkingBudget")
	}
	t.Logf("Content=%q Thinking(len=%d) tokens: in=%d out=%d",
		resp.Content, len(resp.Thinking), resp.InputTokens, resp.OutputTokens)
}

// TestThinkingBudgetGemini sends a per-request ThinkingBudget to Gemini.
// Requires GEMINI_API_KEY; skipped otherwise.
func TestThinkingBudgetGemini(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in -short mode")
	}
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	client, err := llm.NewClient(llm.Config{
		Provider:  "gemini",
		Model:     "gemini-2.5-flash",
		APIKey:    apiKey,
		MaxTokens: 1024,
		Timeout:   60 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	resp, err := client.Complete(context.Background(), llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "What is 7 * 8? Answer with just the number."),
		},
		ThinkingLevel:  llm.ThinkingLow,
		ThinkingBudget: 1024,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if resp.Content == "" {
		t.Error("Content is empty")
	}
	t.Logf("Content=%q Thinking(len=%d) ThinkingTokens=%d tokens: in=%d out=%d",
		resp.Content, len(resp.Thinking), resp.ThinkingTokens, resp.InputTokens, resp.OutputTokens)
}

// TestThinkingBudgetOpenAI tests ThinkingLevel + ReasoningSummary on GPT-5.4.
// GPT-5.4 models use the Responses API and support reasoning effort control.
// Requires OPENAI_API_KEY; skipped otherwise.
func TestThinkingBudgetOpenAI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in -short mode")
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	client, err := llm.NewClient(llm.Config{
		Provider:  "openai",
		Model:     "gpt-5.4-nano",
		APIKey:    apiKey,
		MaxTokens: 2048,
		Timeout:   60 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	resp, err := client.Complete(context.Background(), llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser,
				"A bat and a ball cost $1.10 total. The bat costs $1.00 more than the ball. How much does the ball cost? Show your reasoning."),
		},
		ThinkingLevel:    llm.ThinkingHigh,
		ReasoningSummary: llm.ReasoningSummaryDetailed,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if resp.Content == "" {
		t.Error("Content is empty")
	}
	if resp.Thinking != "" {
		t.Logf("Thinking summary (len=%d): %.200s", len(resp.Thinking), resp.Thinking)
	} else {
		t.Log("Thinking is empty (reasoning may not surface via ReasoningSummary on this model)")
	}
	t.Logf("Content=%q tokens: in=%d out=%d",
		resp.Content, resp.InputTokens, resp.OutputTokens)
}

// TestReasoningSummaryConstants verifies all ReasoningSummary constants exist.
func TestReasoningSummaryConstants(t *testing.T) {
	for _, tc := range []struct {
		name string
		val  llm.ReasoningSummary
		want string
	}{
		{"None", llm.ReasoningSummaryNone, ""},
		{"Auto", llm.ReasoningSummaryAuto, "auto"},
		{"Detailed", llm.ReasoningSummaryDetailed, "detailed"},
		{"Concise", llm.ReasoningSummaryConcise, "concise"},
	} {
		got := string(tc.val)
		if got != tc.want {
			t.Errorf("ReasoningSummary%s = %q, want %q", tc.name, got, tc.want)
		}
		fmt.Printf("  ReasoningSummary%-10s = %q\n", tc.name, got)
	}
}
