// Tutorial 07: Request Logging Middleware
//
// Demonstrates: Config.LogRequests, WithLogging, WithLoggingPrefix,
//               LoggingClient.Stream, Client.Provider, Client.Model
//
// rho/llm supports metadata-only logging (no message content is logged).
// There are two ways to enable it:
//   1. Set LogRequests: true in Config — logging is automatic
//   2. Wrap an existing client manually with WithLogging / WithLoggingPrefix

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
	apiKey := os.Getenv("GEMINI_API_KEY")

	req := llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "What is 7 * 8?"),
		},
	}

	// --- Approach 1: Config-based logging ---
	fmt.Println("=== Approach 1: Config.LogRequests ===")
	{
		cfg := llm.Config{
			Provider:    "gemini",
			Model:       "gemini-2.5-flash",
			APIKey:      apiKey,
			MaxTokens:   100,
			Timeout:     30 * time.Second,
			LogRequests: true, // Logs provider, model, tokens, cost, elapsed time
		}

		client, err := llm.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Printf("Response: %s\n", resp.Content)
		}
		client.Close()
	}
	fmt.Println()

	// --- Approach 2: Manual wrapping with WithLogging ---
	fmt.Println("=== Approach 2: WithLogging ===")
	{
		cfg := llm.Config{
			Provider:  "gemini",
			Model:     "gemini-2.5-flash",
			APIKey:    apiKey,
			MaxTokens: 100,
			Timeout:   30 * time.Second,
		}

		client, err := llm.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Wrap with logging — no prefix
		client = llm.WithLogging(client)

		// You can also inspect which provider/model the client targets
		fmt.Printf("Client: provider=%s, model=%s\n", client.Provider(), client.Model())

		resp, err := client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Printf("Response: %s\n", resp.Content)
		}
		client.Close()
	}
	fmt.Println()

	// --- Approach 3: Manual wrapping with WithLoggingPrefix ---
	fmt.Println("=== Approach 3: WithLoggingPrefix ===")
	{
		cfg := llm.Config{
			Provider:  "gemini",
			Model:     "gemini-2.5-flash",
			APIKey:    apiKey,
			MaxTokens: 100,
			Timeout:   30 * time.Second,
		}

		client, err := llm.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Wrap with a custom prefix for log lines
		client = llm.WithLoggingPrefix(client, "[MyApp]")

		resp, err := client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Printf("Response: %s\n", resp.Content)
		}
		client.Close()
	}
	fmt.Println()

	// --- Approach 4: WithLoggingPrefix + Stream (LoggingClient.Stream) ---
	fmt.Println("=== Approach 4: WithLoggingPrefix + Stream ===")
	{
		cfg := llm.Config{
			Provider:  "gemini",
			Model:     "gemini-2.5-flash",
			APIKey:    apiKey,
			MaxTokens: 100,
			Timeout:   30 * time.Second,
		}

		client, err := llm.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Wrap with prefix — then use Stream instead of Complete
		client = llm.WithLoggingPrefix(client, "[StreamTest]")

		fmt.Print("Streaming: ")
		for event, err := range client.Stream(ctx, req) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nStream error: %v\n", err)
				break
			}
			switch event.Type {
			case llm.EventContent:
				fmt.Print(event.Text)
			case llm.EventDone:
				fmt.Printf("\nDone: reason=%s, input=%d, output=%d\n",
					event.StopReason, event.InputTokens, event.OutputTokens)
			}
		}
		client.Close()
	}
}
