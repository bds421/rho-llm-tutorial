// Tutorial 10: Streaming Tool Use with Error Recovery
//
// Demonstrates: EventToolUse in streaming, event.ToolCall, ToolCall.ID/Name/Input,
//               NewToolResultMessage with isError=true, streaming agentic loop

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

func executeTool(name string, input any) (string, bool) {
	args, ok := input.(map[string]any)
	if !ok {
		return "invalid input format", true
	}

	switch name {
	case "lookup_city":
		city, _ := args["city"].(string)
		city = strings.ToLower(city)
		populations := map[string]string{
			"tokyo":  `{"city":"Tokyo","population":"13.96 million","country":"Japan"}`,
			"paris":  `{"city":"Paris","population":"2.16 million","country":"France"}`,
			"berlin": `{"city":"Berlin","population":"3.75 million","country":"Germany"}`,
		}
		if data, ok := populations[city]; ok {
			return data, false
		}
		// Return error — the model should recover and try a different approach
		return fmt.Sprintf("city %q not found in database", city), true

	default:
		return fmt.Sprintf("unknown tool: %s", name), true
	}
}

func main() {
	ctx := context.Background()

	cfg := llm.Config{
		Provider:  "gemini",
		Model:     "flash",
		APIKey:    os.Getenv("GEMINI_API_KEY"),
		MaxTokens: 512,
		Timeout:   30 * time.Second,
	}

	client, err := llm.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	tools := []llm.Tool{
		{
			Name:        "lookup_city",
			Description: "Look up population data for a city",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{
						"type":        "string",
						"description": "City name",
					},
				},
				"required": []string{"city"},
			},
		},
	}

	req := llm.Request{
		System: "Use the lookup_city tool. If a city is not found, say so. Be concise.",
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "What are the populations of Tokyo and Atlantis?"),
		},
		Tools: tools,
	}

	fmt.Println("=== Streaming Tool Use with Error Recovery ===")
	fmt.Println("Sending initial request (streaming)...")

	// Agentic loop using Stream instead of Complete
	for {
		var contentBuf strings.Builder
		var toolCalls []llm.ToolCall

		fmt.Print("  Stream: ")
		for event, err := range client.Stream(ctx, req) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nStream error: %v\n", err)
				os.Exit(1)
			}
			switch event.Type {
			case llm.EventContent:
				fmt.Print(event.Text)
				contentBuf.WriteString(event.Text)
			case llm.EventToolUse:
				// Tool call received during streaming
				fmt.Printf("[tool:%s(%s)] ", event.ToolCall.Name, formatInput(event.ToolCall.Input))
				toolCalls = append(toolCalls, *event.ToolCall)
			case llm.EventDone:
				fmt.Printf("\n  Done: reason=%s, in=%d, out=%d\n",
					event.StopReason, event.InputTokens, event.OutputTokens)
			}
		}

		// If no tool calls were received, we're done.
		// All providers now return normalized stop reasons (end_turn, tool_use,
		// max_tokens), but checking toolCalls length is still the most robust
		// approach for deciding whether to continue the agentic loop.
		if len(toolCalls) == 0 {
			fmt.Printf("\nFinal answer:\n%s\n", contentBuf.String())
			break
		}

		// Execute tools and build results
		fmt.Printf("  Executing %d tool call(s)...\n", len(toolCalls))
		var results []llm.Message
		for _, tc := range toolCalls {
			output, isError := executeTool(tc.Name, tc.Input)
			if isError {
				fmt.Printf("    %s -> ERROR: %s\n", tc.Name, output)
			} else {
				fmt.Printf("    %s -> %s\n", tc.Name, output)
			}
			// Pass isError=true so the model knows the call failed and can recover
			results = append(results, llm.NewToolResultMessage(tc.ID, output, isError))
		}

		// Append assistant content + tool results for next round.
		// Only append the assistant text if non-empty — Anthropic rejects empty text blocks.
		if contentBuf.Len() > 0 {
			req.Messages = append(req.Messages, llm.NewTextMessage(llm.RoleAssistant, contentBuf.String()))
		}
		req.Messages = append(req.Messages, results...)
	}
}

func formatInput(input any) string {
	b, _ := json.Marshal(input)
	return string(b)
}
