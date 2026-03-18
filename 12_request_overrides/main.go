// Tutorial 12: Per-Request Overrides & Stop Sequences
//
// Demonstrates: Request.Temperature, Request.MaxTokens, Request.StopSequences,
//               Request.Model (overriding Config.Model), Response.StopReason

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

	cfg := llm.Config{
		Provider:  "gemini",
		Model:     "gemini-2.5-flash",
		APIKey:    os.Getenv("GEMINI_API_KEY"),
		MaxTokens: 1024, // default max tokens
		// Temperature: nil means "use provider default, don't send".
		// Set explicitly with ptrFloat64(1.0) to force a value on the wire.
		Timeout: 30 * time.Second,
	}

	client, err := llm.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// --- Test 1: Temperature comparison ---
	fmt.Println("=== Test 1: Temperature 0 vs 1.5 ===")
	prompt := "Give me one random word."

	for _, temp := range []float64{0.0, 1.5} {
		t := temp // capture for pointer
		req := llm.Request{
			Messages: []llm.Message{
				llm.NewTextMessage(llm.RoleUser, prompt),
			},
			Temperature: &t, // *float64 — explicit value sent on the wire
			MaxTokens:   20,
		}

		resp, err := client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error (temp=%.1f): %v\n", temp, err)
			continue
		}
		fmt.Printf("  temp=%.1f -> %s\n", temp, resp.Content)
	}
	fmt.Println()

	// --- Test 2: Per-request MaxTokens override ---
	fmt.Println("=== Test 2: MaxTokens Override ===")
	{
		// Config has MaxTokens=1024, but we override to 10 per-request
		req := llm.Request{
			Messages: []llm.Message{
				llm.NewTextMessage(llm.RoleUser, "Explain general relativity in detail."),
			},
			MaxTokens: 10, // very short — should truncate
		}

		resp, err := client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		} else {
			fmt.Printf("  Response (max 10 tokens): %s\n", resp.Content)
			fmt.Printf("  Stop reason: %s\n", resp.StopReason)
			fmt.Printf("  Output tokens: %d\n", resp.OutputTokens)
		}
	}
	fmt.Println()

	// --- Test 3: Stop sequences ---
	fmt.Println("=== Test 3: Stop Sequences ===")
	{
		req := llm.Request{
			Messages: []llm.Message{
				llm.NewTextMessage(llm.RoleUser, "Count from 1 to 10, separated by commas: 1, 2, 3,"),
			},
			MaxTokens:     100,
			StopSequences: []string{" 7"}, // stop when the model outputs " 7"
		}

		resp, err := client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		} else {
			fmt.Printf("  Response: %s\n", resp.Content)
			fmt.Printf("  Stop reason: %s (should indicate stop sequence hit)\n", resp.StopReason)
		}
	}
	fmt.Println()

	// --- Test 4: Per-request model override ---
	fmt.Println("=== Test 4: Request.Model Override ===")
	{
		// Config.Model is "flash" but we override per-request
		req := llm.Request{
			Model: "gemini-2.5-flash-lite", // override to a different model
			Messages: []llm.Message{
				llm.NewTextMessage(llm.RoleUser, "What model are you? Answer in one word."),
			},
			MaxTokens: 50,
		}

		resp, err := client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		} else {
			fmt.Printf("  Config model: %s\n", client.Model())
			fmt.Printf("  Request model: gemini-2.5-flash-lite\n")
			fmt.Printf("  Response.Model: %s\n", resp.Model)
			fmt.Printf("  Response: %s\n", resp.Content)
		}
	}
}
