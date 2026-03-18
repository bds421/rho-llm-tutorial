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

	tests := []struct {
		name     string
		provider string
		model    string
		apiKey   string
		stream   bool
	}{
		// Cloud providers
		{"Gemini Complete", "gemini", "flash", os.Getenv("GEMINI_API_KEY"), false},
		{"Gemini Stream", "gemini", "flash", os.Getenv("GEMINI_API_KEY"), true},
		{"Anthropic Complete", "anthropic", "haiku", os.Getenv("ANTHROPIC_API_KEY"), false},
		{"Anthropic Stream", "anthropic", "haiku", os.Getenv("ANTHROPIC_API_KEY"), true},
		// Ollama (local, no API key — requires `ollama pull <model>` first)
		{"Ollama Qwen3 Complete", "ollama", "qwen3:4b", "", false},
		{"Ollama Qwen3 Stream", "ollama", "qwen3:4b", "", true},
		{"Ollama Gemma3 Complete", "ollama", "gemma3:4b", "", false},
		{"Ollama Gemma3 Stream", "ollama", "gemma3:4b", "", true},
	}

	prompt := "What is 2+2? Answer with just the number."

	for _, t := range tests {
		fmt.Printf("\n=== %s ===\n", t.name)

		if t.apiKey == "" && t.provider != "ollama" {
			fmt.Printf("Skipped: no API key for %s\n", t.provider)
			continue
		}

		cfg := llm.Config{
			Provider:  t.provider,
			Model:     t.model,
			APIKey:    t.apiKey,
			Timeout:   30 * time.Second,
			MaxTokens: 100,
		}

		client, err := llm.NewClient(cfg)
		if err != nil {
			fmt.Printf("Error creating client: %v\n", err)
			continue
		}

		req := llm.Request{
			Messages: []llm.Message{
				llm.NewTextMessage(llm.RoleUser, prompt),
			},
		}

		start := time.Now()

		if t.stream {
			err = testStream(ctx, client, req)
		} else {
			err = testComplete(ctx, client, req)
		}

		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Time: %v\n", elapsed.Round(time.Millisecond))
		}

		client.Close()
	}

	fmt.Println("\n=== Done ===")
}

func testComplete(ctx context.Context, client llm.Client, req llm.Request) error {
	resp, err := client.Complete(ctx, req)
	if err != nil {
		return err
	}
	fmt.Printf("Response: %s\n", resp.Content)
	fmt.Printf("Tokens: in=%d, out=%d\n", resp.InputTokens, resp.OutputTokens)
	return nil
}

func testStream(ctx context.Context, client llm.Client, req llm.Request) error {
	fmt.Print("Response: ")
	var inputTokens, outputTokens int

	for event, err := range client.Stream(ctx, req) {
		if err != nil {
			return err
		}
		switch event.Type {
		case llm.EventContent:
			fmt.Print(event.Text)
		case llm.EventDone:
			inputTokens = event.InputTokens
			outputTokens = event.OutputTokens
		}
	}

	fmt.Println()
	fmt.Printf("Tokens: in=%d, out=%d\n", inputTokens, outputTokens)
	return nil
}
