// Tutorial 15: Multi-Provider Comparison
//
// Demonstrates: using multiple providers in one program, comparing responses,
//               EstimateCost across providers, provider-agnostic code,
//               thinking/reasoning content from models that think by default

package main

import (
	"context"
	"fmt"
	"log/slog"
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
	// Suppress internal library logs (pool creation, rotation) so the
	// comparison table stays clean.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	})))

	ctx := context.Background()

	prompt := "What is the square root of 144? Answer with just the number."

	// Configure multiple providers — the Request code is identical for all.
	//
	// Token budget note: models with Thinking: true in the registry (Qwen3,
	// DeepSeek-R1, etc.) consume output tokens for internal chain-of-thought
	// before producing the visible answer. A small MaxTokens (e.g. 50) may be
	// entirely consumed by reasoning, leaving nothing for the actual response.
	// The Gemini adapter pads maxOutputTokens automatically; for OpenAI-compat
	// providers (Ollama, Groq) you must set a larger budget yourself.
	providers := []providerConfig{
		{
			name: "Gemini Flash",
			config: llm.Config{
				Provider:  "gemini",
				Model:     "gemini-2.5-flash",
				APIKey:    os.Getenv("GEMINI_API_KEY"),
				MaxTokens: 100,
				Timeout:   30 * time.Second,
			},
		},
		{
			name: "Anthropic Haiku",
			config: llm.Config{
				Provider:  "anthropic",
				Model:     "claude-haiku-4-5-20251001",
				APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
				MaxTokens: 100,
				Timeout:   30 * time.Second,
			},
		},
		{
			name: "Ollama Qwen3:4b",
			config: llm.Config{
				Provider: "ollama",
				Model:    "qwen3:4b",
				// Qwen3 thinks by default (info.Thinking == true). With 100 tokens
				// the reasoning alone would exhaust the budget. 2048 gives enough
				// headroom for chain-of-thought + answer on most prompts.
				MaxTokens: 2048,
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

		// Show thinking content if the model reasoned (Gemini 2.5, Qwen3, etc.)
		if resp.Thinking != "" {
			thinking := resp.Thinking
			if len(thinking) > 72 {
				thinking = thinking[:72] + "..."
			}
			fmt.Printf("  └─ thinking: %s\n", thinking)
		}
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

		// Check if the model reasons intrinsically — useful for adjusting
		// expectations (higher latency, output tokens include reasoning).
		info, _ := llm.GetModelInfo(llm.ResolveModelAlias(p.config.Model))
		tag := ""
		if info.Thinking {
			tag = " [thinks]"
		}
		fmt.Printf("  %s%s: ", p.name, tag)
		var thinkingTokens int
		for event, err := range client.Stream(ctx, req) {
			if err != nil {
				fmt.Printf("[error: %v]", err)
				break
			}
			switch event.Type {
			case llm.EventThinking:
				thinkingTokens += len(event.Thinking) // approximate
			case llm.EventContent:
				fmt.Print(event.Text)
			case llm.EventDone:
				resolvedModel := llm.ResolveModelAlias(p.config.Model)
				cost := llm.EstimateCost(resolvedModel, event.InputTokens, event.OutputTokens)
				fmt.Printf(" (in=%d, out=%d, $%.6f", event.InputTokens, event.OutputTokens, cost)
				if thinkingTokens > 0 {
					fmt.Printf(", thought ~%d chars", thinkingTokens)
				}
				fmt.Println(")")
			}
		}
		client.Close()
	}
}
