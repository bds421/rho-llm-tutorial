// Tutorial 16: Auth Pool Deep Dive
//
// Demonstrates: NewAuthPool, AuthPool.Count/Status/GetCurrent/GetAvailable/
//               MarkFailedByName/MarkSuccessByName, AuthProfile.IsAvailable/
//               MarkUsed/MarkFailed/MarkHealthy, CooldownError, ErrNoAvailableProfiles,
//               NewPooledClient, PooledClient.PoolStatus, key|baseurl pipe syntax
//
// No API keys required — everything runs offline using pool internals and a mock client.

package main

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"os"
	"time"

	"gitlab2024.bds421-cloud.com/bds421/rho/llm"
)

func main() {
	// =========================================================================
	// Step 1: Create an AuthPool and inspect it
	// =========================================================================
	fmt.Println("=== Step 1: AuthPool Creation & Inspection ===")

	keys := []string{
		"sk-key-alpha",
		"sk-key-beta",
		"sk-key-gamma|https://custom-endpoint.example.com/v1", // pipe syntax: key|baseurl
	}

	pool := llm.NewAuthPool("demo", keys)
	fmt.Printf("Pool count: %d\n", pool.Count())
	fmt.Printf("Pool status: %s\n", pool.Status())

	// GetCurrent returns the active profile
	current, ok := pool.GetCurrent()
	if ok {
		fmt.Printf("Current profile: name=%s, key=%s, baseURL=%q\n",
			current.Name, current.APIKey, current.BaseURL)
	}
	fmt.Println()

	// =========================================================================
	// Step 2: AuthProfile lifecycle — IsAvailable, MarkUsed, MarkFailed, MarkHealthy
	// =========================================================================
	fmt.Println("=== Step 2: AuthProfile Lifecycle ===")

	// Get the current profile (copy) — demonstrate IsAvailable and MarkUsed
	profile, _ := pool.GetCurrent()
	fmt.Printf("Profile %q: IsAvailable=%v\n", profile.Name, profile.IsAvailable())

	profile.MarkUsed()
	fmt.Printf("After MarkUsed: LastUsed=%v\n", profile.LastUsed.Format(time.TimeOnly))

	// Mark failed with a cooldown
	profile.MarkFailed(fmt.Errorf("rate limited"), 5*time.Second)
	fmt.Printf("After MarkFailed: IsHealthy=%v, IsAvailable=%v, LastError=%q\n",
		profile.IsHealthy, profile.IsAvailable(), profile.LastError)
	fmt.Printf("  Cooldown until: %v\n", profile.Cooldown.Format(time.TimeOnly))

	// Mark healthy again
	profile.MarkHealthy()
	fmt.Printf("After MarkHealthy: IsHealthy=%v, IsAvailable=%v\n",
		profile.IsHealthy, profile.IsAvailable())
	fmt.Println()

	// =========================================================================
	// Step 3: Pool rotation — GetAvailable, MarkFailedByName, MarkSuccessByName
	// =========================================================================
	fmt.Println("=== Step 3: Pool Rotation ===")

	// Get an available profile from the pool
	available, err := pool.GetAvailable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Available profile: %s\n", available.Name)

	// Mark first profile as failed — pool will rotate to next
	pool.MarkFailedByName(available.Name, fmt.Errorf("auth error"))
	fmt.Printf("After marking %s failed:\n  %s\n", available.Name, pool.Status())

	// Mark it successful to restore
	pool.MarkSuccessByName(available.Name)
	fmt.Printf("After marking %s success:\n  %s\n", available.Name, pool.Status())
	fmt.Println()

	// =========================================================================
	// Step 4: Exhaust all profiles → trigger ErrNoAvailableProfiles + CooldownError
	// =========================================================================
	fmt.Println("=== Step 4: ErrNoAvailableProfiles & CooldownError ===")

	// Create a small pool and exhaust it
	smallPool := llm.NewAuthPool("test", []string{"key-1", "key-2"})

	// Mark all profiles as failed with cooldowns
	smallPool.MarkFailedByName("test-1", fmt.Errorf("rate limited"))
	smallPool.MarkFailedByName("test-2", fmt.Errorf("overloaded"))

	// Now GetAvailable should fail
	_, err = smallPool.GetAvailable()
	if err != nil {
		fmt.Printf("GetAvailable error: %v\n", err)

		// Check for ErrNoAvailableProfiles
		if errors.Is(err, llm.ErrNoAvailableProfiles) {
			fmt.Println("  -> errors.Is(err, ErrNoAvailableProfiles) = true")
		}

		// Extract CooldownError with errors.As
		var cooldownErr *llm.CooldownError
		if errors.As(err, &cooldownErr) {
			fmt.Printf("  -> CooldownError.Error(): %s\n", cooldownErr.Error())
			fmt.Printf("  -> CooldownError.Wait: %v\n", cooldownErr.Wait)

			// Unwrap should give ErrNoAvailableProfiles
			unwrapped := cooldownErr.Unwrap()
			fmt.Printf("  -> CooldownError.Unwrap(): %v\n", unwrapped)
			fmt.Printf("  -> errors.Is(Unwrap(), ErrNoAvailableProfiles) = %v\n",
				errors.Is(unwrapped, llm.ErrNoAvailableProfiles))
		}

		// Round-trip: wrap and re-check
		wrapped := fmt.Errorf("pool failure: %w", err)
		fmt.Printf("  -> errors.Is(wrapped, ErrNoAvailableProfiles) = %v\n",
			errors.Is(wrapped, llm.ErrNoAvailableProfiles))
	}
	fmt.Println()

	// =========================================================================
	// Step 5: NewPooledClient with a mock client function
	// =========================================================================
	fmt.Println("=== Step 5: NewPooledClient (mock) ===")

	cfg := llm.Config{
		Provider:  "demo",
		Model:     "mock-model",
		MaxTokens: 100,
	}

	mockClientFunc := func(profile llm.AuthProfile) (llm.Client, error) {
		return &mockClient{provider: "demo", model: "mock-model"}, nil
	}

	pc, err := llm.NewPooledClient(cfg, []string{"key-a", "key-b"}, mockClientFunc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewPooledClient error: %v\n", err)
		os.Exit(1)
	}
	defer pc.Close()

	fmt.Printf("PooledClient provider: %s\n", pc.Provider())
	fmt.Printf("PooledClient model: %s\n", pc.Model())
	fmt.Printf("PoolStatus: %s\n", pc.PoolStatus())

	// Exercise PooledClient.Complete() directly (not behind Client interface)
	ctx := context.Background()
	resp, err := pc.Complete(ctx, llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "test"),
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Complete error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Complete response: %s\n", resp.Content)

	// Exercise PooledClient.Stream() directly
	fmt.Print("Stream events: ")
	for event, err := range pc.Stream(ctx, llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, "test"),
		},
	}) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Stream error: %v\n", err)
			break
		}
		if event.Type == llm.EventDone {
			fmt.Printf("done (reason=%s)\n", event.StopReason)
		}
	}
}

// mockClient implements llm.Client with no-op methods for offline testing.
type mockClient struct {
	provider string
	model    string
}

func (m *mockClient) Complete(_ context.Context, _ llm.Request) (*llm.Response, error) {
	return &llm.Response{Content: "mock response"}, nil
}

func (m *mockClient) Stream(_ context.Context, _ llm.Request) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) {
		yield(llm.StreamEvent{Type: llm.EventDone, StopReason: "end_turn"}, nil)
	}
}

func (m *mockClient) Provider() string { return m.provider }
func (m *mockClient) Model() string    { return m.model }
func (m *mockClient) Close() error     { return nil }
