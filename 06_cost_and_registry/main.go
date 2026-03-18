// Tutorial 06: Model Registry, Cost Estimation & Aliases
//
// Demonstrates: ResolveModelAlias, GetModelInfo, ModelInfo fields
//               (ID, Provider, ContextWindow, InputPricePer1M, OutputPricePer1M,
//                SupportsThinking, Thinking, MaxTokens),
//               EstimateCost, ProviderForModel, GetDefaultModel
//
// The registry contains metadata for all known models — pricing, context windows,
// thinking support, and aliases. This tutorial is purely informational and makes
// no API calls.

package main

import (
	"fmt"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

func main() {
	// --- Step 1: Resolve short aliases to full model IDs ---
	aliases := []string{"opus", "sonnet", "haiku", "flash", "grok", "grok-code", "gemini-pro", "gpt", "mistral-small", "groq", "codestral"}
	fmt.Println("=== Alias Resolution ===")
	for _, alias := range aliases {
		resolved := llm.ResolveModelAlias(alias)
		fmt.Printf("  %-20s -> %s\n", alias, resolved)
	}
	fmt.Println()

	// --- Step 2: Query per-model metadata ---
	fmt.Println("=== Model Info ===")
	models := []string{
		"claude-opus-4-6",
		"claude-sonnet-4-6",
		"gemini-2.5-flash",
		"grok-4-fast-non-reasoning",
		"gpt-5.4-nano",
		"mistral-small-2603",
	}
	for _, model := range models {
		info, ok := llm.GetModelInfo(model)
		if !ok {
			fmt.Printf("  %s: not found in registry\n", model)
			continue
		}
		fmt.Printf("  %s\n", info.ID)
		fmt.Printf("    Provider:          %s\n", info.Provider)
		fmt.Printf("    Context window:    %d tokens\n", info.ContextWindow)
		fmt.Printf("    Max output tokens: %d\n", info.MaxTokens)
		fmt.Printf("    Input pricing:     $%.2f / 1M tokens\n", info.InputPricePer1M)
		fmt.Printf("    Output pricing:    $%.2f / 1M tokens\n", info.OutputPricePer1M)
		fmt.Printf("    Supports thinking: %v\n", info.SupportsThinking)
		fmt.Printf("    Intrinsic reason:  %v\n", info.Thinking)
		fmt.Println()
	}

	// --- Step 3: Estimate cost from token counts ---
	fmt.Println("=== Cost Estimation ===")
	type scenario struct {
		model  string
		input  int
		output int
	}
	scenarios := []scenario{
		{"claude-sonnet-4-6", 1000, 500},
		{"claude-opus-4-6", 10000, 2000},
		{"gemini-2.5-flash", 5000, 1000},
	}
	for _, s := range scenarios {
		cost := llm.EstimateCost(llm.CostInput{
			Model:        s.model,
			InputTokens:  s.input,
			OutputTokens: s.output,
		})
		fmt.Printf("  %s (%d in, %d out): $%.6f\n", s.model, s.input, s.output, cost)
	}
	fmt.Println()

	// --- Step 4: Detect provider from model ID ---
	fmt.Println("=== Provider Detection ===")
	for _, model := range []string{"gemini-2.5-flash", "claude-sonnet-4-6", "gpt-5.4"} {
		provider := llm.ProviderForModel(model)
		fmt.Printf("  %-30s -> provider: %s\n", model, provider)
	}
	fmt.Println()

	// --- Step 5: Get default model for a provider ---
	fmt.Println("=== Default Models ===")
	for _, prov := range []string{"anthropic", "gemini", "xai", "openai", "groq", "mistral"} {
		model := llm.GetDefaultModel(prov)
		fmt.Printf("  %-12s -> %s\n", prov, model)
	}
}
