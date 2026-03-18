// Tutorial 03: Tool Use (Function Calling)
//
// Demonstrates: Tool, ToolCall, Request.Tools, Response.StopReason,
//               Response.ToolCalls, NewToolResultMessage (success & error),
//               RoleAssistant, the agentic tool-use loop pattern
//
// Tool use lets the model invoke functions you define. When the model wants
// to call a tool, resp.StopReason == "tool_use". You execute the tool locally
// and feed the result back via NewToolResultMessage.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

// executeTool dispatches tool calls to local implementations.
func executeTool(name string, input any) (string, bool) {
	args, ok := input.(map[string]any)
	if !ok {
		return "invalid input format", true // isError=true
	}

	switch name {
	case "calculate":
		op, _ := args["operation"].(string)
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		var result float64
		switch op {
		case "add":
			result = a + b
		case "multiply":
			result = a * b
		case "power":
			result = math.Pow(a, b)
		default:
			return fmt.Sprintf("unknown operation: %s", op), true
		}
		return fmt.Sprintf("%.4f", result), false

	case "get_weather":
		location, _ := args["location"].(string)
		// Simulated weather lookup
		return fmt.Sprintf(`{"location":"%s","temp_c":22,"condition":"sunny"}`, location), false

	default:
		return fmt.Sprintf("unknown tool: %s", name), true
	}
}

func main() {
	ctx := context.Background()

	cfg := llm.Config{
		Provider: "gemini",
		Model:    "gemini-2.5-flash",
		APIKey:   os.Getenv("GEMINI_API_KEY"),
		Timeout:  30 * time.Second,
	}

	client, err := llm.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// --- Step 1: Define tools via JSON Schema ---
	// Each Tool has a Name, Description, and InputSchema (map[string]any).
	tools := []llm.Tool{
		{
			Name:        "calculate",
			Description: "Perform a math operation on two numbers",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operation": map[string]any{
						"type":        "string",
						"enum":        []string{"add", "multiply", "power"},
						"description": "The operation to perform",
					},
					"a": map[string]any{"type": "number", "description": "First operand"},
					"b": map[string]any{"type": "number", "description": "Second operand"},
				},
				"required": []string{"operation", "a", "b"},
			},
		},
		{
			Name:        "get_weather",
			Description: "Get the current weather for a city",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "City name, e.g. 'Berlin'",
					},
				},
				"required": []string{"location"},
			},
		},
	}

	// --- Step 2: Send the initial request with tools ---
	req := llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser,
				"What is 2^10, and what's the weather in Tokyo? Use the tools."),
		},
		Tools: tools,
	}

	fmt.Println("Sending initial request...")
	resp, err := client.Complete(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// --- Step 3: The agentic loop ---
	// Keep looping as long as the model wants to call tools.
	for resp.StopReason == "tool_use" {
		fmt.Printf("Model requested %d tool call(s)\n", len(resp.ToolCalls))

		// Execute each tool call and collect results
		var results []llm.Message
		for _, tc := range resp.ToolCalls {
			fmt.Printf("  -> %s(%s)\n", tc.Name, formatInput(tc.Input))

			output, isError := executeTool(tc.Name, tc.Input)

			// NewToolResultMessage links the result back to the request via tc.ID.
			// Pass isError=true if the tool execution failed — the model can recover.
			results = append(results, llm.NewToolResultMessage(tc.ID, output, isError))
		}

		// Append the assistant's intermediate response (its "thought")
		req.Messages = append(req.Messages, llm.NewTextMessage(llm.RoleAssistant, resp.Content))
		// Append all tool results
		req.Messages = append(req.Messages, results...)

		fmt.Println("Sending tool results back...")
		resp, err = client.Complete(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// --- Step 4: Final response ---
	fmt.Printf("\nFinal answer:\n%s\n", resp.Content)
	fmt.Printf("Tokens: input=%d, output=%d\n", resp.InputTokens, resp.OutputTokens)
}

func formatInput(input any) string {
	b, _ := json.Marshal(input)
	return string(b)
}
