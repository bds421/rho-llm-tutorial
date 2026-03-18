// Tutorial 02: Streaming Responses
//
// Demonstrates: Client.Stream, StreamEvent, EventType constants
//               (EventContent, EventDone), event.Text, event.StopReason,
//               event.InputTokens, event.OutputTokens
//
// Stream() returns a Go 1.23 iterator (iter.Seq2[StreamEvent, error])
// that yields events as the model generates tokens. This lets you display
// partial output in real time rather than waiting for the full response.

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
		Model:     "haiku", // alias for claude-haiku-4-5
		APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
		MaxTokens: 512,
		Timeout:   30 * time.Second,
	}

	client, err := llm.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	req := llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "Write a short poem about Go iterators."),
		},
	}

	fmt.Println("Streaming response:")
	fmt.Println("---")

	// Stream returns an iterator — use Go 1.23 range-over-func syntax.
	// Each iteration yields a StreamEvent and an optional error.
	for event, err := range client.Stream(ctx, req) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nStream error: %v\n", err)
			break // break aborts early; the iterator cleans up the HTTP connection
		}

		switch event.Type {
		case llm.EventContent:
			// A chunk of generated text. Concatenate all chunks for the full response.
			fmt.Print(event.Text)

		case llm.EventDone:
			// Final metadata — stop reason and token usage.
			// StopReason is "end_turn", "tool_use", or "max_tokens".
			fmt.Printf("\n---\nDone: reason=%s, input=%d, output=%d\n",
				event.StopReason, event.InputTokens, event.OutputTokens)
		}
	}
	// Note: EventDone is optional metadata. If the connection drops,
	// the iterator may exhaust without it. Handle iterator exhaustion
	// as the authoritative "stream ended" signal.
}
