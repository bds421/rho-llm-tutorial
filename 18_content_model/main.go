// Tutorial 18: Content Model Internals
//
// Demonstrates: ContentPart, ContentType constants (ContentText, ContentImage,
//               ContentToolUse, ContentToolResult), ImageSource,
//               ToolCall.ThoughtSignature
//
// No API keys required — all content structures are built and inspected locally.

package main

import (
	"encoding/json"
	"fmt"

	"gitlab2024.bds421-cloud.com/bds421/rho/llm"
)

func main() {
	// =========================================================================
	// Step 1: Inspect NewTextMessage output → ContentText constant
	// =========================================================================
	fmt.Println("=== Step 1: NewTextMessage → ContentPart ===")

	msg := llm.NewTextMessage(llm.RoleUser, "Hello, world!")
	fmt.Printf("Role: %s\n", msg.Role)
	fmt.Printf("Parts: %d\n", len(msg.Content))
	for i, part := range msg.Content {
		fmt.Printf("  Part[%d]: Type=%s, Text=%q\n", i, part.Type, part.Text)
	}
	fmt.Println()

	// =========================================================================
	// Step 2: Build a multimodal Message with ContentImage + ImageSource
	// =========================================================================
	fmt.Println("=== Step 2: Multimodal Message (text + image) ===")

	multimodal := llm.Message{
		Role: llm.RoleUser,
		Content: []llm.ContentPart{
			{
				Type: llm.ContentText,
				Text: "What is in this image?",
			},
			{
				Type: llm.ContentImage,
				Source: &llm.ImageSource{
					Type:      "base64",
					MediaType: "image/png",
					Data:      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==", // 1x1 red pixel
				},
			},
		},
	}

	fmt.Printf("Role: %s\n", multimodal.Role)
	fmt.Printf("Parts: %d\n", len(multimodal.Content))
	for i, part := range multimodal.Content {
		fmt.Printf("  Part[%d]: Type=%s", i, part.Type)
		if part.Text != "" {
			fmt.Printf(", Text=%q", part.Text)
		}
		if part.Source != nil {
			fmt.Printf(", Source.Type=%s, Source.MediaType=%s, Source.Data=%d bytes",
				part.Source.Type, part.Source.MediaType, len(part.Source.Data))
		}
		fmt.Println()
	}
	fmt.Println()

	// =========================================================================
	// Step 3: Inspect NewToolResultMessage → ContentToolResult
	// =========================================================================
	fmt.Println("=== Step 3: NewToolResultMessage → ContentToolResult ===")

	toolResult := llm.NewToolResultMessage("call_123", `{"temperature": 22}`, false)
	fmt.Printf("Role: %s\n", toolResult.Role)
	for i, part := range toolResult.Content {
		fmt.Printf("  Part[%d]: Type=%s, ToolResultID=%s, Content=%s, IsError=%v\n",
			i, part.Type, part.ToolResultID, part.ToolResultContent, part.IsError)
	}
	fmt.Println()

	// =========================================================================
	// Step 4: ContentToolUse part with ThoughtSignature
	// =========================================================================
	fmt.Println("=== Step 4: ContentToolUse + ThoughtSignature ===")

	toolUsePart := llm.ContentPart{
		Type:             llm.ContentToolUse,
		ToolUseID:        "call_456",
		ToolName:         "get_weather",
		ToolInput:        map[string]string{"city": "Paris"},
		ThoughtSignature: "gemini3-sig-abc123",
	}

	fmt.Printf("Type: %s\n", toolUsePart.Type)
	fmt.Printf("ToolUseID: %s\n", toolUsePart.ToolUseID)
	fmt.Printf("ToolName: %s\n", toolUsePart.ToolName)
	fmt.Printf("ThoughtSignature: %s\n", toolUsePart.ThoughtSignature)

	// Also show ToolCall.ThoughtSignature
	tc := llm.ToolCall{
		ID:               "call_789",
		Name:             "search",
		Input:            map[string]string{"q": "go iterators"},
		ThoughtSignature: "gemini3-sig-xyz789",
	}
	fmt.Printf("\nToolCall.ThoughtSignature: %s\n", tc.ThoughtSignature)
	fmt.Println()

	// =========================================================================
	// Step 5: All 4 ContentType constants
	// =========================================================================
	fmt.Println("=== Step 5: ContentType Constants ===")

	types := []struct {
		name  string
		value llm.ContentType
	}{
		{"ContentText", llm.ContentText},
		{"ContentImage", llm.ContentImage},
		{"ContentToolUse", llm.ContentToolUse},
		{"ContentToolResult", llm.ContentToolResult},
	}

	for _, t := range types {
		fmt.Printf("  %-20s = %q\n", t.name, t.value)
	}
	fmt.Println()

	// =========================================================================
	// Step 6: JSON round-trip to show serialization
	// =========================================================================
	fmt.Println("=== Step 6: JSON Serialization ===")

	data, _ := json.MarshalIndent(multimodal, "", "  ")
	fmt.Println(string(data))
}
