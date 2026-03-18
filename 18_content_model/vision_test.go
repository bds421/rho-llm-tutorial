package main

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

// TestGeminiVisionDescribesImage sends a real image (golden retriever cropped
// from the LLM-Image-Classification test set) to Gemini and verifies the
// response mentions "dog" or "retriever".
//
// Image source: github.com/robert-mcdermott/LLM-Image-Classification
//
// Requires GEMINI_API_KEY; skipped otherwise.
func TestGeminiVisionDescribesImage(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	// Read test image from testdata/
	imgData, err := os.ReadFile("testdata/dog.png")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(imgData)

	client, err := llm.NewClient(llm.Config{
		Provider:  "gemini",
		Model:     "gemini-2.5-flash",
		APIKey:    apiKey,
		MaxTokens: 100,
		Timeout:   30 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	resp, err := client.Complete(context.Background(), llm.Request{
		Messages: []llm.Message{
			{
				Role: llm.RoleUser,
				Content: []llm.ContentPart{
					{Type: llm.ContentImage, Source: &llm.ImageSource{
						Type:      "base64",
						MediaType: "image/png",
						Data:      b64,
					}},
					{Type: llm.ContentText, Text: "What animal is in this image? Answer in one sentence."},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	lower := strings.ToLower(resp.Content)
	if !strings.Contains(lower, "dog") && !strings.Contains(lower, "retriever") {
		t.Errorf("expected response to mention 'dog' or 'retriever', got: %s", resp.Content)
	}

	cost := llm.EstimateCost(llm.CostInput{
		Model:        "gemini-2.5-flash",
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
	})
	t.Logf("Content=%q, in=%d, out=%d, cost=$%.6f", resp.Content, resp.InputTokens, resp.OutputTokens, cost)
}

// TestAnthropicVisionDescribesImage sends the same image to Anthropic Haiku
// and verifies the response mentions "dog" or "retriever".
//
// Requires ANTHROPIC_API_KEY; skipped otherwise.
func TestAnthropicVisionDescribesImage(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	imgData, err := os.ReadFile("testdata/dog.png")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(imgData)

	client, err := llm.NewClient(llm.Config{
		Provider:  "anthropic",
		Model:     "claude-haiku-4-5-20251001",
		APIKey:    apiKey,
		MaxTokens: 100,
		Timeout:   30 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	resp, err := client.Complete(context.Background(), llm.Request{
		Messages: []llm.Message{
			{
				Role: llm.RoleUser,
				Content: []llm.ContentPart{
					{Type: llm.ContentImage, Source: &llm.ImageSource{
						Type:      "base64",
						MediaType: "image/png",
						Data:      b64,
					}},
					{Type: llm.ContentText, Text: "What animal is in this image? Answer in one sentence."},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	lower := strings.ToLower(resp.Content)
	if !strings.Contains(lower, "dog") && !strings.Contains(lower, "retriever") {
		t.Errorf("expected response to mention 'dog' or 'retriever', got: %s", resp.Content)
	}

	cost := llm.EstimateCost(llm.CostInput{
		Model:        "claude-haiku-4-5-20251001",
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
	})
	t.Logf("Content=%q, in=%d, out=%d, cost=$%.6f", resp.Content, resp.InputTokens, resp.OutputTokens, cost)
}
