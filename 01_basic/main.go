// Tutorial 01: Basic Complete Request
//
// Demonstrates: Config, NewClient, Client.Complete, Client.Close,
//               Request, Response, Message, NewTextMessage,
//               RoleUser, resp.Content, resp.InputTokens, resp.OutputTokens
//
// This is the simplest possible usage of rho/llm — configure a provider,
// send a single message, and print the response.

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"gitlab2024.bds421-cloud.com/bds421/rho/llm"
	_ "gitlab2024.bds421-cloud.com/bds421/rho/llm/provider" // required: register all provider adapters
)

func main() {
	ctx := context.Background()

	// --- Step 1: Configure the client ---
	// Config holds all connection settings. Only Provider and Model are required
	// for cloud providers (plus APIKey). Local providers like Ollama need no key.
	cfg := llm.Config{
		Provider:  "gemini",
		Model:     "flash",                    // alias — resolves to "gemini-2.5-flash"
		APIKey:    os.Getenv("GEMINI_API_KEY"), // from environment
		MaxTokens: 256,                        // max output tokens (default: 8192)
		Timeout:   30 * time.Second,           // HTTP timeout (default: 120s)
	}

	// --- Step 2: Create a client ---
	// NewClient looks up the registered provider adapter and returns a Client.
	// The blank import of llm/provider above is what makes providers available.
	client, err := llm.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close() // always close to release HTTP resources

	// --- Step 3: Build a request ---
	// A Request contains a slice of Messages. Each message has a Role and content.
	// NewTextMessage is the constructor for simple text messages.
	req := llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "Explain quantum entanglement in one sentence."),
		},
	}

	// --- Step 4: Send and receive ---
	// Complete sends the request and blocks until the full response is available.
	resp, err := client.Complete(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// --- Step 5: Use the response ---
	// Response contains the generated text and token usage statistics.
	fmt.Println("Response:", resp.Content)
	fmt.Printf("Stop reason: %s\n", resp.StopReason)
	fmt.Printf("Tokens: input=%d, output=%d\n", resp.InputTokens, resp.OutputTokens)
}
