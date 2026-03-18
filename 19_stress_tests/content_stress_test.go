package stress_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	llm "github.com/bds421/rho-llm"
)

func TestContentPart_LargeTextPayload(t *testing.T) {
	// 1MB text
	large := strings.Repeat("A", 1024*1024)
	msg := llm.NewTextMessage(llm.RoleUser, large)

	// JSON round-trip
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded llm.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(decoded.Content) != 1 {
		t.Fatalf("Content len = %d, want 1", len(decoded.Content))
	}
	if decoded.Content[0].Text != large {
		t.Error("large text not preserved through JSON round-trip")
	}
}

func TestContentPart_ManyParts(t *testing.T) {
	parts := make([]llm.ContentPart, 10000)
	for i := range parts {
		parts[i] = llm.ContentPart{
			Type: llm.ContentText,
			Text: fmt.Sprintf("part-%d", i),
		}
	}

	msg := llm.Message{Role: llm.RoleUser, Content: parts}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded llm.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(decoded.Content) != 10000 {
		t.Errorf("Content len = %d, want 10000", len(decoded.Content))
	}
	if decoded.Content[9999].Text != "part-9999" {
		t.Errorf("last part text = %q, want %q", decoded.Content[9999].Text, "part-9999")
	}
}

func TestContentPart_DeepToolCallChain(t *testing.T) {
	// Simulate a 50-turn tool conversation with 100 messages
	messages := make([]llm.Message, 0, 100)

	for turn := 0; turn < 50; turn++ {
		// User message
		messages = append(messages, llm.NewTextMessage(llm.RoleUser, fmt.Sprintf("turn %d", turn)))

		// Assistant response with tool call
		resp := &llm.Response{
			Content: fmt.Sprintf("thinking about turn %d", turn),
			ToolCalls: []llm.ToolCall{
				{
					ID:    fmt.Sprintf("call-%d", turn),
					Name:  "lookup",
					Input: map[string]interface{}{"query": fmt.Sprintf("q%d", turn)},
				},
			},
		}
		messages = append(messages, llm.NewAssistantMessage(resp))
	}

	if len(messages) != 100 {
		t.Errorf("message count = %d, want 100", len(messages))
	}

	// JSON round-trip
	data, err := json.Marshal(messages)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded []llm.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(decoded) != 100 {
		t.Errorf("decoded count = %d, want 100", len(decoded))
	}
}

func TestNewAssistantMessage_ManyToolCalls(t *testing.T) {
	toolCalls := make([]llm.ToolCall, 100)
	for i := range toolCalls {
		toolCalls[i] = llm.ToolCall{
			ID:    fmt.Sprintf("call-%d", i),
			Name:  fmt.Sprintf("tool-%d", i),
			Input: map[string]interface{}{"n": i},
		}
	}

	resp := &llm.Response{
		Content:   "text content",
		ToolCalls: toolCalls,
	}

	msg := llm.NewAssistantMessage(resp)

	// 1 text + 100 tool_use = 101 content parts
	if len(msg.Content) != 101 {
		t.Errorf("Content len = %d, want 101", len(msg.Content))
	}

	// First part should be text
	if msg.Content[0].Type != llm.ContentText {
		t.Errorf("first part type = %v, want %v", msg.Content[0].Type, llm.ContentText)
	}
	if msg.Content[0].Text != "text content" {
		t.Errorf("first part text = %q, want %q", msg.Content[0].Text, "text content")
	}

	// Last part should be tool_use
	last := msg.Content[100]
	if last.Type != llm.ContentToolUse {
		t.Errorf("last part type = %v, want %v", last.Type, llm.ContentToolUse)
	}
	if last.ToolUseID != "call-99" {
		t.Errorf("last tool ID = %q, want %q", last.ToolUseID, "call-99")
	}
}

func TestNewAssistantMessage_EmptyResponse(t *testing.T) {
	resp := &llm.Response{
		Content:   "",
		ToolCalls: nil,
	}

	msg := llm.NewAssistantMessage(resp)

	if msg.Role != llm.RoleAssistant {
		t.Errorf("Role = %v, want %v", msg.Role, llm.RoleAssistant)
	}
	if len(msg.Content) != 0 {
		t.Errorf("Content len = %d, want 0 for empty response", len(msg.Content))
	}
}

func TestNewToolResultMessage_LargeResult(t *testing.T) {
	// 1MB JSON result string
	large := strings.Repeat(`{"key":"value"},`, 64*1024) // ~1MB

	msg := llm.NewToolResultMessage("call-1", large, false)

	if len(msg.Content) != 1 {
		t.Fatalf("Content len = %d, want 1", len(msg.Content))
	}
	if msg.Content[0].ToolResultContent != large {
		t.Error("large result not preserved")
	}
	if msg.Content[0].ToolResultID != "call-1" {
		t.Errorf("ToolResultID = %q, want %q", msg.Content[0].ToolResultID, "call-1")
	}
}

func TestImageSource_LargeData(t *testing.T) {
	// 5MB of raw data → base64 encoded
	raw := make([]byte, 5*1024*1024)
	for i := range raw {
		raw[i] = byte(i % 256)
	}
	b64 := base64.StdEncoding.EncodeToString(raw)

	part := llm.ContentPart{
		Type: llm.ContentImage,
		Source: &llm.ImageSource{
			Type:      "base64",
			MediaType: "image/png",
			Data:      b64,
		},
	}

	// Should not panic
	msg := llm.Message{Role: llm.RoleUser, Content: []llm.ContentPart{part}}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded llm.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Content[0].Source.Data != b64 {
		t.Error("large image data not preserved through JSON round-trip")
	}
}

func BenchmarkNewTextMessage(b *testing.B) {
	text := strings.Repeat("hello ", 170) // ~1KB
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = llm.NewTextMessage(llm.RoleUser, text)
	}
}

func BenchmarkNewAssistantMessage_WithToolCalls(b *testing.B) {
	toolCalls := make([]llm.ToolCall, 10)
	for i := range toolCalls {
		toolCalls[i] = llm.ToolCall{
			ID:    fmt.Sprintf("call-%d", i),
			Name:  "tool",
			Input: map[string]interface{}{"n": i},
		}
	}
	resp := &llm.Response{
		Content:   "text",
		ToolCalls: toolCalls,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = llm.NewAssistantMessage(resp)
	}
}
