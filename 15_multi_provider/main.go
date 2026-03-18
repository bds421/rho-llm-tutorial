// Tutorial 15: Multi-Provider Comparison
//
// Demonstrates: using multiple providers in one program, comparing responses,
//               EstimateCost across providers, provider-agnostic code

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

type providerConfig struct {
	name   string
	config llm.Config
}

func main() {
	ctx := context.Background()

	prompt := "What is the square root of 144? Answer with just the number."

	// Configure multiple providers — the Request code is identical for all
	providers := []providerConfig{
		{
			name: "Gemini Flash",
			config: llm.Config{
				Provider:  "gemini",
				Model:     "flash",
				APIKey:    os.Getenv("GEMINI_API_KEY"),
				MaxTokens: 50,
				Timeout:   30 * time.Second,
			},
		},
		{
			name: "Anthropic Haiku",
			config: llm.Config{
				Provider:  "anthropic",
				Model:     "haiku",
				APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
				MaxTokens: 50,
				Timeout:   30 * time.Second,
			},
		},
		{
			name: "Ollama Qwen3:4b",
			config: llm.Config{
				Provider:  "ollama",
				Model:     "qwen3:4b",
				MaxTokens: 50,
				Timeout:   60 * time.Second,
			},
		},
	}

	fmt.Printf("Prompt: %q\n\n", prompt)

	// The same Request works across all providers
	req := llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, prompt),
		},
	}

	fmt.Printf("%-20s %-10s %-8s %-8s %-12s %-10s\n",
		"Provider", "Response", "In Tok", "Out Tok", "Cost", "Latency")
	fmt.Println("-------------------- ---------- -------- -------- ------------ ----------")

	for _, p := range providers {
		if p.config.APIKey == "" && !llm.IsNoAuthProvider(p.config.Provider) {
			fmt.Printf("%-20s %-10s (skipped — no API key)\n", p.name, "—")
			continue
		}

		client, err := llm.NewClient(p.config)
		if err != nil {
			fmt.Printf("%-20s error: %v\n", p.name, err)
			continue
		}

		start := time.Now()
		resp, err := client.Complete(ctx, req)
		elapsed := time.Since(start)
		client.Close()

		if err != nil {
			fmt.Printf("%-20s error: %v\n", p.name, err)
			continue
		}

		// EstimateCost works with the resolved model name
		resolvedModel := llm.ResolveModelAlias(p.config.Model)
		cost := llm.EstimateCost(resolvedModel, resp.InputTokens, resp.OutputTokens)

		// Truncate response for table display
		content := resp.Content
		if len(content) > 8 {
			content = content[:8] + ".."
		}

		fmt.Printf("%-20s %-10s %-8d %-8d $%-11.6f %v\n",
			p.name, content, resp.InputTokens, resp.OutputTokens, cost,
			elapsed.Round(time.Millisecond))
	}

	fmt.Println()

	// --- Streaming comparison: same prompt, different providers ---
	fmt.Println("=== Streaming Comparison ===")
	for _, p := range providers {
		if p.config.APIKey == "" && !llm.IsNoAuthProvider(p.config.Provider) {
			continue
		}

		client, err := llm.NewClient(p.config)
		if err != nil {
			continue
		}

		fmt.Printf("  %s: ", p.name)
		for event, err := range client.Stream(ctx, req) {
			if err != nil {
				fmt.Printf("[error: %v]", err)
				break
			}
			switch event.Type {
			case llm.EventContent:
				fmt.Print(event.Text)
			case llm.EventDone:
				fmt.Printf(" (in=%d, out=%d)\n", event.InputTokens, event.OutputTokens)
			}
		}
		client.Close()
	}
}
