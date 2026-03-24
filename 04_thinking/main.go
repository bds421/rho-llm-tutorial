// Tutorial 04: Extended Thinking & Reasoning
//
// Demonstrates: Config.ThinkingLevel, ThinkingNone/ThinkingLow/ThinkingMedium/ThinkingHigh,
//               Config.ProviderName, Client.Provider(),
//               GetModelInfo, ModelInfo.SupportsThinking, ModelInfo.Thinking,
//               Response.Thinking, EventThinking, event.Thinking,
//               ThinkingBudgetTokens, Request.ThinkingBudget, Request.ThinkingLevel,
//               ReasoningSummary (auto/detailed/concise)
//
// Many modern models support reasoning (chain-of-thought). There are two flavors:
//   1. API-controlled thinking budgets (e.g. Anthropic) — opt-in via ThinkingLevel
//   2. Intrinsic reasoning models (e.g. DeepSeek-R1) — always reason natively
//
// If you pass ThinkingLevel to a non-reasoning model, it is silently ignored.

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

func main() {
	ctx := context.Background()

	// --- Step 1: Check model thinking capabilities via the registry ---
	modelID := "claude-opus-4-6"
	info, ok := llm.GetModelInfo(modelID)
	if !ok {
		fmt.Fprintf(os.Stderr, "Model %q not found in registry\n", modelID)
		os.Exit(1)
	}

	fmt.Printf("Model: %s\n", info.ID)
	fmt.Printf("  SupportsThinking (API-controlled budgets): %v\n", info.SupportsThinking)
	fmt.Printf("  Thinking (intrinsic reasoning):            %v\n", info.Thinking)
	fmt.Printf("  Context window: %d tokens\n", info.ContextWindow)
	fmt.Println()

	// Show all ThinkingLevel constants
	fmt.Println("ThinkingLevel constants:")
	for _, level := range []llm.ThinkingLevel{llm.ThinkingNone, llm.ThinkingLow, llm.ThinkingMedium, llm.ThinkingHigh} {
		if level == "" {
			fmt.Printf("  ThinkingNone   = %q\n", level)
		} else {
			fmt.Printf("  %-14s = %q\n", level, level)
		}
	}
	fmt.Println()

	// --- Step 1b: ThinkingNone + Config.ProviderName ---
	fmt.Println("=== ThinkingNone + ProviderName ===")
	{
		noneCfg := llm.Config{
			Provider:      "anthropic",
			Model:         "claude-haiku-4-5-20251001",
			APIKey:        os.Getenv("ANTHROPIC_API_KEY"),
			MaxTokens:     100,
			ThinkingLevel: llm.ThinkingNone, // Explicitly no thinking
			ProviderName:  "anthropic-via-proxy",
			Timeout:       30 * time.Second,
		}

		noneClient, err := llm.NewClient(noneCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		// Provider() returns the overridden name
		fmt.Printf("Provider (overridden): %s\n", noneClient.Provider())

		noneResp, err := noneClient.Complete(ctx, llm.Request{
			Messages: []llm.Message{
				llm.NewTextMessage(llm.RoleUser, "Say hello in one word."),
			},
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Printf("Response: %s\n", noneResp.Content)
			fmt.Printf("Thinking (should be empty): %q\n", noneResp.Thinking)
		}
		noneClient.Close()
	}
	fmt.Println()

	// --- Step 2: Configure with extended thinking enabled ---
	cfg := llm.Config{
		Provider:      "anthropic",
		Model:         "claude-sonnet-4-6",
		APIKey:        os.Getenv("ANTHROPIC_API_KEY"),
		MaxTokens:     16000,                          // required when thinking is enabled
		ThinkingLevel: llm.ThinkingHigh,                // ThinkingLow / ThinkingMedium / ThinkingHigh
		Timeout:       120 * time.Second,
	}

	client, err := llm.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	req := llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser,
				"A bat and a ball cost $1.10 in total. The bat costs $1.00 more than the ball. How much does the ball cost? Show your reasoning."),
		},
	}

	// --- Step 3a: Synchronous — thinking is in resp.Thinking ---
	fmt.Println("=== Synchronous (Complete) ===")
	resp, err := client.Complete(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if resp.Thinking != "" {
		fmt.Printf("Thinking:\n%s\n\n", resp.Thinking)
	}
	fmt.Printf("Answer:\n%s\n", resp.Content)
	fmt.Printf("Tokens: input=%d, output=%d\n\n", resp.InputTokens, resp.OutputTokens)

	// --- Step 3b: Streaming — thinking arrives via EventThinking ---
	fmt.Println("=== Streaming ===")
	fmt.Println("[Thinking]")

	for event, err := range client.Stream(ctx, req) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nStream error: %v\n", err)
			break
		}
		switch event.Type {
		case llm.EventThinking:
			// Extended thinking output — arrives before the final answer
			fmt.Print(event.Thinking)
		case llm.EventContent:
			fmt.Print(event.Text)
		case llm.EventDone:
			fmt.Printf("\n\nDone: reason=%s, input=%d, output=%d\n",
				event.StopReason, event.InputTokens, event.OutputTokens)
		}
	}
	fmt.Println()

	// --- Step 4: ThinkingBudgetTokens — inspect default budgets (v0.2.2+) ---
	// ThinkingBudgetTokens resolves a ThinkingLevel to its default token count.
	// Pass customBudget > 0 to override the level default.
	fmt.Println("=== ThinkingBudgetTokens ===")
	for _, level := range []llm.ThinkingLevel{
		llm.ThinkingNone, llm.ThinkingMinimal, llm.ThinkingLow,
		llm.ThinkingMedium, llm.ThinkingHigh, llm.ThinkingXHigh,
	} {
		tokens := llm.ThinkingBudgetTokens(level, 0) // 0 = use level default
		label := string(level)
		if label == "" {
			label = "(none)"
		}
		fmt.Printf("  %-10s → %6d tokens\n", label, tokens)
	}
	// Custom budget overrides the level default
	custom := llm.ThinkingBudgetTokens(llm.ThinkingHigh, 5000)
	fmt.Printf("  high+custom=5000 → %d tokens\n\n", custom)

	// --- Step 5: Per-request ThinkingBudget override (Anthropic) ---
	// Request.ThinkingBudget and Request.ThinkingLevel let you override
	// the client-level Config on a per-request basis.
	fmt.Println("=== Per-request ThinkingBudget ===")
	budgetReq := llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "What is 7 * 8? Answer with just the number."),
		},
		ThinkingLevel:  llm.ThinkingLow, // per-request override
		ThinkingBudget: 1024,            // custom token budget for this request only
	}

	budgetResp, err := client.Complete(ctx, budgetReq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	} else {
		fmt.Printf("Answer: %s\n", budgetResp.Content)
		if budgetResp.Thinking != "" {
			fmt.Printf("Thinking (len=%d): %.80s...\n", len(budgetResp.Thinking), budgetResp.Thinking)
		}
		fmt.Printf("Tokens: input=%d, output=%d\n", budgetResp.InputTokens, budgetResp.OutputTokens)
	}
	fmt.Println()

	// --- Step 6: ReasoningSummary constants (v0.2.2+) ---
	// ReasoningSummary controls reasoning summary text in responses.
	// Used with OpenAI Responses API (GPT-5 family). Available on Request.
	fmt.Println("=== ReasoningSummary constants ===")
	for _, rs := range []struct {
		name string
		val  llm.ReasoningSummary
	}{
		{"ReasoningSummaryNone", llm.ReasoningSummaryNone},
		{"ReasoningSummaryAuto", llm.ReasoningSummaryAuto},
		{"ReasoningSummaryDetailed", llm.ReasoningSummaryDetailed},
		{"ReasoningSummaryConcise", llm.ReasoningSummaryConcise},
	} {
		fmt.Printf("  %-28s = %q\n", rs.name, rs.val)
	}
}
