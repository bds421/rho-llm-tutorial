// Tutorial 05: Structured Error Handling
//
// Demonstrates: APIError, IsRateLimited, IsOverloaded, IsAuthError,
//               IsContextLength, IsRetryable, Backoff
//
// All API errors are returned as *APIError with an HTTP status code,
// enabling reliable classification for retry logic.

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

func main() {
	ctx := context.Background()

	// Use a deliberately bad API key to trigger an auth error
	cfg := llm.Config{
		Provider:  "anthropic",
		Model:     "claude-haiku-4-5-20251001",
		APIKey:    "sk-invalid-key-for-demo",
		MaxTokens: 100,
		Timeout:   10 * time.Second,
	}

	client, err := llm.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	req := llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "Hello"),
		},
	}

	// --- Attempt the request with manual retry logic ---
	maxRetries := 3
	for attempt := range maxRetries {
		fmt.Printf("Attempt %d/%d...\n", attempt+1, maxRetries)

		resp, err := client.Complete(ctx, req)
		if err != nil {
			// --- Classify the error using helper functions ---
			switch {
			case llm.IsAuthError(err):
				// 401/403 — bad API key. No point retrying.
				fmt.Printf("  Auth error: %v\n", err)
				fmt.Println("  -> Check your API key. Not retrying.")
				os.Exit(1)

			case llm.IsRateLimited(err):
				// 429 — too many requests. Back off and retry.
				delay := llm.Backoff(attempt, 1*time.Second, 30*time.Second)
				fmt.Printf("  Rate limited. Backing off %v...\n", delay)
				time.Sleep(delay)
				continue

			case llm.IsOverloaded(err):
				// 503 — server busy. Retry later.
				delay := llm.Backoff(attempt, 2*time.Second, 30*time.Second)
				fmt.Printf("  Server overloaded. Backing off %v...\n", delay)
				time.Sleep(delay)
				continue

			case llm.IsContextLength(err):
				// 400 — input too long. Truncate, don't retry.
				fmt.Printf("  Context length exceeded: %v\n", err)
				fmt.Println("  -> Reduce input size.")
				os.Exit(1)

			case llm.IsRetryable(err):
				// Any retryable error (429, 503, 500, 502, 408)
				delay := llm.Backoff(attempt, 1*time.Second, 30*time.Second)
				fmt.Printf("  Retryable error: %v. Backing off %v...\n", err, delay)
				time.Sleep(delay)
				continue

			default:
				// Non-retryable error — inspect the underlying *APIError if present
				var apiErr *llm.APIError
				if errors.As(err, &apiErr) {
					fmt.Printf("  API error (HTTP %d): %s [provider=%s, retryable=%v]\n",
						apiErr.StatusCode, apiErr.Message, apiErr.Provider, apiErr.Retryable)
				} else {
					fmt.Printf("  Unexpected error: %v\n", err)
				}
				os.Exit(1)
			}
		}

		// Success
		fmt.Printf("  Response: %s\n", resp.Content)
		fmt.Printf("  Tokens: input=%d, output=%d\n", resp.InputTokens, resp.OutputTokens)
		return
	}

	fmt.Println("All retries exhausted.")
}
