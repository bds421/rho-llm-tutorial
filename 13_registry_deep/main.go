// Tutorial 13: Deep Registry Exploration
//
// Demonstrates: GetAvailableModels, DefaultConfig, ModelInfo.Label,
//               ModelInfo.NoToolSupport, ModelInfo.ThoughtSignature,
//               comprehensive model enumeration

package main

import (
	"fmt"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

func main() {
	// --- Test 1: DefaultConfig ---
	fmt.Println("=== DefaultConfig ===")
	{
		cfg := llm.DefaultConfig()
		fmt.Printf("  Provider:      %s\n", cfg.Provider)
		fmt.Printf("  Model:         %s\n", cfg.Model)
		fmt.Printf("  MaxTokens:     %d\n", cfg.MaxTokens)
		fmt.Printf("  Temperature:   %.1f\n", cfg.Temperature)
		fmt.Printf("  Timeout:       %v\n", cfg.Timeout)
		fmt.Printf("  ThinkingLevel: %q\n", cfg.ThinkingLevel)
	}
	fmt.Println()

	// --- Test 2: GetAvailableModels per provider ---
	fmt.Println("=== Available Models Per Provider ===")
	providers := []string{"anthropic", "gemini", "xai", "openai", "groq", "mistral"}
	for _, prov := range providers {
		models := llm.GetAvailableModels(prov)
		fmt.Printf("  %s (%d models):\n", prov, len(models))
		for _, m := range models {
			info, ok := llm.GetModelInfo(m)
			marker := ""
			if ok {
				if info.SupportsThinking {
					marker += " [thinking]"
				}
				if info.Thinking {
					marker += " [intrinsic-reasoning]"
				}
				if info.NoToolSupport {
					marker += " [no-tools]"
				}
				if info.ThoughtSignature {
					marker += " [thought-signature]"
				}
				if info.Label != "" {
					marker += fmt.Sprintf(" (%s)", info.Label)
				}
			}
			fmt.Printf("    - %s%s\n", m, marker)
		}
		fmt.Println()
	}

	// --- Test 3: Models with thinking support ---
	fmt.Println("=== Models with Thinking Support ===")
	for _, prov := range providers {
		for _, m := range llm.GetAvailableModels(prov) {
			info, ok := llm.GetModelInfo(m)
			if ok && (info.SupportsThinking || info.Thinking) {
				thinkType := "API-controlled"
				if info.Thinking {
					thinkType = "intrinsic"
				}
				fmt.Printf("  %-40s %s (%s)\n", m, prov, thinkType)
			}
		}
	}
	fmt.Println()

	// --- Test 4: Models with ThoughtSignature ---
	fmt.Println("=== Models with ThoughtSignature ===")
	for _, prov := range providers {
		for _, m := range llm.GetAvailableModels(prov) {
			info, ok := llm.GetModelInfo(m)
			if ok && info.ThoughtSignature {
				fmt.Printf("  %-40s %s\n", m, prov)
			}
		}
	}
	fmt.Println()

	// --- Test 5: Models without tool support ---
	fmt.Println("=== Models Without Tool Support ===")
	for _, prov := range providers {
		for _, m := range llm.GetAvailableModels(prov) {
			info, ok := llm.GetModelInfo(m)
			if ok && info.NoToolSupport {
				fmt.Printf("  %-40s %s\n", m, prov)
			}
		}
	}
}
