// Tutorial 18: Content Model Internals
//
// Demonstrates: ContentPart, ContentType constants (ContentText, ContentImage,
//               ContentToolUse, ContentToolResult), ImageSource,
//               NewImageMessage, ValidateImageSource, ToolCall.ThoughtSignature
//
// Steps 1-6 require no API keys — all content structures are built and inspected locally.
// Step 7 sends a real image to a vision-capable model (requires GEMINI_API_KEY).

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
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
	fmt.Println()

	// =========================================================================
	// Step 7: NewImageMessage helper + ValidateImageSource
	// =========================================================================
	fmt.Println("=== Step 7: NewImageMessage + ValidateImageSource ===")

	// 1x1 red pixel PNG
	pixelData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	imgMsg := llm.NewImageMessage(llm.RoleUser, "image/png", pixelData)
	fmt.Printf("NewImageMessage: Role=%s, Parts=%d, Type=%s, MediaType=%s\n",
		imgMsg.Role, len(imgMsg.Content), imgMsg.Content[0].Type, imgMsg.Content[0].Source.MediaType)

	// Validation: valid image passes
	if err := llm.ValidateImageSource(imgMsg.Content[0]); err != nil {
		fmt.Printf("  Unexpected validation error: %v\n", err)
	} else {
		fmt.Println("  ValidateImageSource: OK")
	}

	// Validation: invalid media type
	badPart := llm.ContentPart{
		Type:   llm.ContentImage,
		Source: &llm.ImageSource{Type: "base64", MediaType: "text/plain", Data: "abc"},
	}
	if err := llm.ValidateImageSource(badPart); err != nil {
		fmt.Printf("  ValidateImageSource (text/plain): %v\n", err)
	}
	fmt.Println()

	// =========================================================================
	// Step 8: Live API call — send image to a vision model (optional)
	// =========================================================================
	fmt.Println("=== Step 8: Live Vision API Call (optional) ===")

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("  Skipped — set GEMINI_API_KEY to enable")
		return
	}

	// Encode the 1x1 red pixel as base64
	redPixelPNG, _ := base64.StdEncoding.DecodeString(pixelData)
	_ = redPixelPNG // already have pixelData as base64

	cfg := llm.Config{
		Provider:  "gemini",
		Model:     "gemini-2.5-flash",
		APIKey:    apiKey,
		MaxTokens: 256,
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		fmt.Printf("  NewClient error: %v\n", err)
		return
	}
	defer client.Close()

	req := llm.Request{
		Messages: []llm.Message{{
			Role: llm.RoleUser,
			Content: []llm.ContentPart{
				{Type: llm.ContentText, Text: "Describe this image in one sentence."},
				{Type: llm.ContentImage, Source: &llm.ImageSource{
					Type: "base64", MediaType: "image/png", Data: pixelData,
				}},
			},
		}},
	}

	resp, err := client.Complete(context.Background(), req)
	if err != nil {
		fmt.Printf("  Complete error: %v\n", err)
		return
	}

	fmt.Printf("  Model: %s\n", resp.Model)
	fmt.Printf("  Response: %s\n", resp.Content)
	fmt.Printf("  Tokens: in=%d out=%d\n", resp.InputTokens, resp.OutputTokens)
	fmt.Printf("  Cost: $%.6f\n", llm.EstimateCost(cfg.Model, resp.InputTokens, resp.OutputTokens))
}
