// Tutorial 09: System Messages & Multi-Turn Conversations
//
// Demonstrates: RoleSystem, Request.System, RoleAssistant, RoleUser,
//               NewAssistantMessage, multi-turn conversation flow,
//               Response.ID, Response.Model

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"gitlab2024.bds421-cloud.com/bds421/rho/llm"
	_ "gitlab2024.bds421-cloud.com/bds421/rho/llm/provider"
)

func main() {
	ctx := context.Background()

	cfg := llm.Config{
		Provider:  "anthropic",
		Model:     "haiku",
		APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
		MaxTokens: 256,
		Timeout:   30 * time.Second,
	}

	client, err := llm.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// --- Approach 1: System prompt via Request.System field ---
	fmt.Println("=== Approach 1: Request.System field ===")
	{
		req := llm.Request{
			System: "You are a pirate. Always respond in pirate speak. Keep answers under 30 words.",
			Messages: []llm.Message{
				llm.NewTextMessage(llm.RoleUser, "What is the capital of France?"),
			},
		}

		resp, err := client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Response: %s\n", resp.Content)
		fmt.Printf("Response ID: %s\n", resp.ID)
		fmt.Printf("Response Model: %s\n", resp.Model)
	}
	fmt.Println()

	// --- Approach 2: RoleSystem in message array ---
	// Both Anthropic and Gemini adapters automatically promote RoleSystem
	// messages to the provider's native system parameter (top-level "system"
	// for Anthropic, systemInstruction for Gemini). Request.System is the
	// simpler approach; RoleSystem in the messages array works equally well.
	fmt.Println("=== Approach 2: RoleSystem message (Gemini) ===")
	{
		geminiCfg := llm.Config{
			Provider:  "gemini",
			Model:     "flash",
			APIKey:    os.Getenv("GEMINI_API_KEY"),
			MaxTokens: 256,
			Timeout:   30 * time.Second,
		}
		geminiClient, err := llm.NewClient(geminiCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer geminiClient.Close()

		req := llm.Request{
			Messages: []llm.Message{
				llm.NewTextMessage(llm.RoleSystem, "You are a Shakespearean actor. Always respond in iambic pentameter. Keep answers under 30 words."),
				llm.NewTextMessage(llm.RoleUser, "What is the capital of France?"),
			},
		}

		resp, err := geminiClient.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Response: %s\n", resp.Content)
	}
	fmt.Println()

	// --- Multi-turn conversation ---
	fmt.Println("=== Multi-Turn Conversation ===")
	{
		messages := []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "My name is Alice. Remember it."),
		}

		// Turn 1
		req := llm.Request{
			System:   "You are a helpful assistant. Keep answers under 20 words.",
			Messages: messages,
		}
		resp, err := client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Turn 1 - User:      My name is Alice. Remember it.\n")
		fmt.Printf("Turn 1 - Assistant:  %s\n", resp.Content)

		// Append assistant response and new user message for turn 2
		// NewAssistantMessage builds a Message from a Response, preserving
		// both text content and any tool calls (important for tool-use loops).
		messages = append(messages, llm.NewAssistantMessage(resp))
		messages = append(messages, llm.NewTextMessage(llm.RoleUser, "What is my name?"))

		// Turn 2
		req.Messages = messages
		resp, err = client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Turn 2 - User:      What is my name?\n")
		fmt.Printf("Turn 2 - Assistant:  %s\n", resp.Content)

		// Append and continue for turn 3
		messages = append(messages, llm.NewAssistantMessage(resp))
		messages = append(messages, llm.NewTextMessage(llm.RoleUser, "Spell it backwards."))

		req.Messages = messages
		resp, err = client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Turn 3 - User:      Spell it backwards.\n")
		fmt.Printf("Turn 3 - Assistant:  %s\n", resp.Content)
		fmt.Printf("\nTotal turns: 3, Final token count: input=%d, output=%d\n",
			resp.InputTokens, resp.OutputTokens)
	}
}
