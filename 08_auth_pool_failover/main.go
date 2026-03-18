// Tutorial 08: Auth Pool Rotation & Multi-Key Failover
//
// Demonstrates: NewClientWithKeys, per-profile endpoints (API_KEY|BASE_URL),
//               Config.BaseURL, Config.AuthHeader, Config.ProviderName,
//               Config.Temperature, the "custom" provider for OpenAI-compat endpoints
//
// NewClientWithKeys creates a pooled client that automatically rotates between
// API keys on failure. The rotation engine is thread-safe and uses exponential
// backoff with jitter. Error classification:
//   - Transient (429, 503, 502): backoff + rotate
//   - Auth errors (401, 403): key permanently disabled, rotate
//   - Bad request (400): return immediately (request is broken, not the key)

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

	// --- Example 1: Multi-key failover with a single provider ---
	fmt.Println("=== Example 1: Multi-Key Anthropic Failover ===")
	{
		cfg := llm.Config{
			Provider:    "anthropic",
			Model:       "claude-sonnet-4-6",
			MaxTokens:   256,
			Temperature: 0.7, // sampling temperature (default: 1.0)
			Timeout:     120 * time.Second,
		}

		// Provide multiple API keys. If one is rate-limited, the pool
		// seamlessly rotates to the next available key.
		keys := []string{
			os.Getenv("ANTHROPIC_API_KEY"),
			os.Getenv("ANTHROPIC_API_KEY_BACKUP"),
		}

		client, err := llm.NewClientWithKeys(cfg, keys)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			// In production you'd handle this; here we continue to next example
		} else {
			req := llm.Request{
				Messages: []llm.Message{
					llm.NewTextMessage(llm.RoleUser, "What is the capital of France?"),
				},
			}

			resp, err := client.Complete(ctx, req)
			if err != nil {
				fmt.Printf("Error (all keys failed): %v\n", err)
			} else {
				fmt.Printf("Response: %s\n", resp.Content)
				cost := llm.EstimateCost("claude-sonnet-4-6", resp.InputTokens, resp.OutputTokens)
				fmt.Printf("Cost: $%.6f\n", cost)
			}
			client.Close()
		}
	}
	fmt.Println()

	// --- Example 2: Per-profile endpoints (heterogeneous failover) ---
	fmt.Println("=== Example 2: Per-Profile Endpoints ===")
	{
		cfg := llm.Config{
			Provider:  "openai",
			Model:     "gpt-4o",
			MaxTokens: 256,
			Timeout:   30 * time.Second,
		}

		// Keys can include a custom BaseURL using the "API_KEY|BASE_URL" format.
		// This enables failover across entirely different backends.
		keys := []string{
			os.Getenv("OPENAI_PRIMARY_KEY"),                       // uses default openai endpoint
			os.Getenv("OPENAI_BACKUP_KEY"),                        // same provider, different key
			os.Getenv("AZURE_OPENAI_KEY") + "|https://my-azure-proxy.example.com/v1", // custom endpoint
		}

		client, err := llm.NewClientWithKeys(cfg, keys)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			req := llm.Request{
				Messages: []llm.Message{
					llm.NewTextMessage(llm.RoleUser, "Say hello."),
				},
			}

			resp, err := client.Complete(ctx, req)
			if err != nil {
				fmt.Printf("All profiles failed: %v\n", err)
			} else {
				fmt.Printf("Response: %s\n", resp.Content)
			}
			client.Close()
		}
	}
	fmt.Println()

	// --- Example 3: Custom OpenAI-compatible endpoint ---
	fmt.Println("=== Example 3: Custom Provider ===")
	{
		// The "custom" provider lets you connect to any OpenAI-compatible API.
		cfg := llm.Config{
			Provider: "custom",
			BaseURL:  "http://localhost:8000/v1", // e.g. a local vLLM instance
			Model:    "my-local-model",
		}

		client, err := llm.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Printf("Custom client created: provider=%s, model=%s\n",
				client.Provider(), client.Model())
			client.Close()
		}
	}
	fmt.Println()

	// --- Example 4: Ollama (local, no API key) ---
	fmt.Println("=== Example 4: Ollama (Local) ===")
	{
		cfg := llm.Config{
			Provider: "ollama",
			Model:    "qwen3:4b", // or "llama3", "mistral", "phi3", etc.
			Timeout:  60 * time.Second,
		}

		client, err := llm.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			req := llm.Request{
				Messages: []llm.Message{
					llm.NewTextMessage(llm.RoleUser, "What is 2+2? Answer with just the number."),
				},
			}

			resp, err := client.Complete(ctx, req)
			if err != nil {
				fmt.Printf("Ollama error: %v\n", err)
			} else {
				fmt.Printf("Response: %s\n", resp.Content)
			}
			client.Close()
		}
	}
}
