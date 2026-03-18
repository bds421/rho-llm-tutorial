// Tutorial 11: Early Stream Abort & EventError
//
// Demonstrates: break in stream loop (early abort with cleanup),
//               EventError handling, context cancellation mid-stream

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
		Provider:  "anthropic",
		Model:     "claude-haiku-4-5-20251001",
		APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
		MaxTokens: 1024,
		Timeout:   30 * time.Second,
	}

	client, err := llm.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// --- Test 1: Early abort with break ---
	// Use break to stop after N tokens. The iterator cleans up the HTTP connection.
	fmt.Println("=== Test 1: Early Stream Abort (break after 50 chars) ===")
	{
		req := llm.Request{
			Messages: []llm.Message{
				llm.NewTextMessage(llm.RoleUser, "Write a 500-word essay about the history of computing."),
			},
		}

		charCount := 0
		for event, err := range client.Stream(ctx, req) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nStream error: %v\n", err)
				break
			}
			switch event.Type {
			case llm.EventContent:
				fmt.Print(event.Text)
				charCount += len(event.Text)
				if charCount > 50 {
					fmt.Println("\n  [ABORTED after 50 chars — break cleans up HTTP connection]")
					goto aborted // break only exits switch; use goto to exit range loop
				}
			case llm.EventError:
				// EventError is emitted when an error occurs mid-stream
				fmt.Printf("\n  EventError: %v\n", event.Error)
			case llm.EventDone:
				fmt.Printf("\n  Done: reason=%s\n", event.StopReason)
			}
		}
	aborted:
		fmt.Printf("  Total chars received before abort: %d\n", charCount)
	}
	fmt.Println()

	// --- Test 2: Context cancellation mid-stream ---
	fmt.Println("=== Test 2: Context Cancellation ===")
	{
		cancelCtx, cancel := context.WithCancel(ctx)

		req := llm.Request{
			Messages: []llm.Message{
				llm.NewTextMessage(llm.RoleUser, "Count from 1 to 100, one number per line."),
			},
		}

		lineCount := 0
		for event, err := range client.Stream(cancelCtx, req) {
			if err != nil {
				fmt.Printf("\n  Stream error after cancel: %v\n", err)
				break
			}
			switch event.Type {
			case llm.EventContent:
				fmt.Print(event.Text)
				lineCount++
				if lineCount > 3 {
					fmt.Println("\n  [Cancelling context...]")
					cancel()
					// The next iteration should return an error
				}
			case llm.EventDone:
				fmt.Printf("\n  Done: reason=%s\n", event.StopReason)
			}
		}
		cancel() // ensure cancel is called even if loop exits normally
		fmt.Printf("  Lines received: %d\n", lineCount)
	}
	fmt.Println()

	// --- Test 3: Timeout mid-stream ---
	fmt.Println("=== Test 3: Short Timeout ===")
	{
		shortCfg := llm.Config{
			Provider:  "anthropic",
			Model:     "claude-haiku-4-5-20251001",
			APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
			MaxTokens: 4096,
			Timeout:   2 * time.Second, // very short timeout
		}

		shortClient, err := llm.NewClient(shortCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer shortClient.Close()

		req := llm.Request{
			Messages: []llm.Message{
				llm.NewTextMessage(llm.RoleUser, "Write a very long detailed essay about every planet in the solar system."),
			},
		}

		charCount := 0
		timedOut := false
		for event, err := range shortClient.Stream(ctx, req) {
			if err != nil {
				fmt.Printf("\n  Timeout/error: %v\n", err)
				timedOut = true
				break
			}
			if event.Type == llm.EventContent {
				charCount += len(event.Text)
			}
			if event.Type == llm.EventDone {
				fmt.Printf("  Completed normally: %s\n", event.StopReason)
			}
		}
		fmt.Printf("  Chars received: %d, timed out: %v\n", charCount, timedOut)
	}
}
