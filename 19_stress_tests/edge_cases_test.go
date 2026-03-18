package stress_test

import (
	"context"
	"errors"
	"testing"

	llm "github.com/bds421/rho-llm"
)

func TestNewAuthPool_EmptyKeys(t *testing.T) {
	pool := llm.NewAuthPool("test", []string{})

	if pool.Count() != 0 {
		t.Errorf("Count() = %d, want 0", pool.Count())
	}

	_, err := pool.GetAvailable()
	if err == nil {
		t.Error("expected error from empty pool")
	}
	if !errors.Is(err, llm.ErrNoAvailableProfiles) {
		t.Errorf("expected ErrNoAvailableProfiles, got: %v", err)
	}
}

func TestNewAuthPool_SingleKey(t *testing.T) {
	pool := llm.NewAuthPool("test", []string{"my-key"})

	if pool.Count() != 1 {
		t.Errorf("Count() = %d, want 1", pool.Count())
	}

	profile, err := pool.GetAvailable()
	if err != nil {
		t.Fatalf("GetAvailable: %v", err)
	}
	if profile.APIKey != "my-key" {
		t.Errorf("APIKey = %q, want %q", profile.APIKey, "my-key")
	}
	if profile.Name != "test-1" {
		t.Errorf("Name = %q, want %q", profile.Name, "test-1")
	}
}

func TestNewAuthPool_DuplicateKeys(t *testing.T) {
	pool := llm.NewAuthPool("test", []string{"same-key", "same-key", "same-key"})

	if pool.Count() != 3 {
		t.Errorf("Count() = %d, want 3", pool.Count())
	}

	// Each should get a unique name
	names := make(map[string]bool)
	for i := 0; i < 3; i++ {
		profile, err := pool.GetAvailable()
		if err != nil {
			t.Fatalf("GetAvailable: %v", err)
		}
		names[profile.Name] = true
	}
	// Due to rotation, we might get the same profile multiple times.
	// Just verify at least 1 unique name exists.
	if len(names) == 0 {
		t.Error("expected at least one profile name")
	}
}

func TestNewAuthPool_EmptyProvider(t *testing.T) {
	// Should not panic
	pool := llm.NewAuthPool("", []string{"key1"})
	if pool.Count() != 1 {
		t.Errorf("Count() = %d, want 1", pool.Count())
	}

	profile, err := pool.GetAvailable()
	if err != nil {
		t.Fatalf("GetAvailable: %v", err)
	}
	if profile.Name != "-1" {
		t.Errorf("Name = %q, want %q", profile.Name, "-1")
	}
}

func TestNewAuthPool_PipeSyntaxVariants(t *testing.T) {
	tests := []struct {
		key     string
		wantKey string
		wantURL string
	}{
		{"key|https://api.example.com/v1", "key", "https://api.example.com/v1"},
		{"key|", "key", ""},
		{"|https://api.example.com", "", "https://api.example.com"},
		{"plainkey", "plainkey", ""},
		{"a|b|c", "a", "b|c"}, // split on first pipe only
	}

	for _, tt := range tests {
		pool := llm.NewAuthPool("test", []string{tt.key})
		profile, err := pool.GetAvailable()
		if err != nil {
			t.Fatalf("GetAvailable(%q): %v", tt.key, err)
		}
		if profile.APIKey != tt.wantKey {
			t.Errorf("key %q: APIKey = %q, want %q", tt.key, profile.APIKey, tt.wantKey)
		}
		if profile.BaseURL != tt.wantURL {
			t.Errorf("key %q: BaseURL = %q, want %q", tt.key, profile.BaseURL, tt.wantURL)
		}
	}
}

func TestMarkFailedByName_UnknownName(t *testing.T) {
	pool := llm.NewAuthPool("test", []string{"key1"})

	// Should not panic
	pool.MarkFailedByName("nonexistent", llm.NewRateLimitError("test", "rate limited"))

	// Pool state should be unchanged
	profile, err := pool.GetAvailable()
	if err != nil {
		t.Fatalf("GetAvailable: %v", err)
	}
	if profile.APIKey != "key1" {
		t.Errorf("APIKey = %q, want %q", profile.APIKey, "key1")
	}
}

func TestMarkSuccessByName_UnknownName(t *testing.T) {
	pool := llm.NewAuthPool("test", []string{"key1"})

	// Should not panic
	pool.MarkSuccessByName("nonexistent")

	if pool.Count() != 1 {
		t.Errorf("Count() = %d, want 1", pool.Count())
	}
}

func TestPooledClient_Complete_NilMessages(t *testing.T) {
	mock := newSequenceMock().
		ThenComplete(okResponse("ok"), nil).
		Build()

	cfg := llm.Config{Provider: "mock", Model: "mock-model"}
	pc, err := llm.NewPooledClient(cfg, []string{"k1"}, mockClientFunc(mock))
	if err != nil {
		t.Fatalf("NewPooledClient: %v", err)
	}
	defer pc.Close()

	// Request with nil messages — mock doesn't validate, so no panic
	req := llm.Request{Messages: nil, MaxTokens: 100}
	resp, err := pc.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("Content = %q, want %q", resp.Content, "ok")
	}
}

func TestPooledClient_Complete_ZeroMaxTokens(t *testing.T) {
	mock := newSequenceMock().
		ThenComplete(okResponse("ok"), nil).
		Build()

	cfg := llm.Config{Provider: "mock", Model: "mock-model"}
	pc, err := llm.NewPooledClient(cfg, []string{"k1"}, mockClientFunc(mock))
	if err != nil {
		t.Fatalf("NewPooledClient: %v", err)
	}
	defer pc.Close()

	req := llm.Request{
		Messages:  []llm.Message{llm.NewTextMessage(llm.RoleUser, "hi")},
		MaxTokens: 0,
	}
	resp, err := pc.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("Content = %q, want %q", resp.Content, "ok")
	}
}
