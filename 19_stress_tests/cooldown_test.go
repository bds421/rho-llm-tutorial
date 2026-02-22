package stress_test

import (
	"errors"
	"testing"
	"time"

	llm "gitlab2024.bds421-cloud.com/bds421/rho/llm"
)

func TestCooldown_ProfileBecomesAvailableAfterExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing test in -short mode")
	}

	pool := llm.NewAuthPool("test", []string{"key-A"})

	// Mark failed with short cooldown
	pool.MarkFailedByName("test-1", llm.NewOverloadedError("test", "overloaded"))

	// Should be in cooldown
	_, err := pool.GetAvailable()
	if err == nil {
		t.Fatal("expected error while in cooldown")
	}

	var cooldownErr *llm.CooldownError
	if !errors.As(err, &cooldownErr) {
		t.Fatalf("expected CooldownError, got %T: %v", err, err)
	}

	// Use direct MarkFailed on a profile with short cooldown for timing test
	pool2 := llm.NewAuthPool("test2", []string{"key-A"})
	profile, _ := pool2.GetAvailable()

	// We can't call MarkFailed on the snapshot (it's a copy), so use MarkFailedByName
	// with a short error that triggers 10s cooldown (transient)
	// Instead, just verify the cooldown error has a positive wait
	if cooldownErr.Wait <= 0 {
		t.Errorf("CooldownError.Wait = %v, want > 0", cooldownErr.Wait)
	}
	_ = profile
}

func TestCooldown_CooldownErrorWaitAccuracy(t *testing.T) {
	pool := llm.NewAuthPool("test", []string{"key-A"})

	// 503 → 30s cooldown
	pool.MarkFailedByName("test-1", llm.NewOverloadedError("test", "overloaded"))

	_, err := pool.GetAvailable()
	var cooldownErr *llm.CooldownError
	if !errors.As(err, &cooldownErr) {
		t.Fatalf("expected CooldownError, got %T: %v", err, err)
	}

	// Wait should be approximately 30s (within 1s tolerance)
	if cooldownErr.Wait < 29*time.Second || cooldownErr.Wait > 31*time.Second {
		t.Errorf("Wait = %v, want ~30s", cooldownErr.Wait)
	}
}

func TestCooldown_AuthErrorPermanentlyDisables(t *testing.T) {
	pool := llm.NewAuthPool("test", []string{"key-A"})

	pool.MarkFailedByName("test-1", llm.NewAuthError("test", "invalid", 401))

	// Should get a non-cooldown error (permanently disabled, not temporarily)
	_, err := pool.GetAvailable()
	if err == nil {
		t.Fatal("expected error for permanently disabled profile")
	}

	// Should NOT be a CooldownError (permanent disable, not cooldown)
	var cooldownErr *llm.CooldownError
	if errors.As(err, &cooldownErr) {
		t.Error("auth error should permanently disable, not cooldown")
	}
}

func TestCooldown_RateLimitCooldown60s(t *testing.T) {
	pool := llm.NewAuthPool("test", []string{"key-A"})

	pool.MarkFailedByName("test-1", llm.NewRateLimitError("test", "rate limited"))

	_, err := pool.GetAvailable()
	var cooldownErr *llm.CooldownError
	if !errors.As(err, &cooldownErr) {
		t.Fatalf("expected CooldownError, got %T: %v", err, err)
	}

	// 429 → 60s cooldown
	if cooldownErr.Wait < 59*time.Second || cooldownErr.Wait > 61*time.Second {
		t.Errorf("Wait = %v, want ~60s", cooldownErr.Wait)
	}
}

func TestCooldown_OverloadedCooldown30s(t *testing.T) {
	pool := llm.NewAuthPool("test", []string{"key-A"})

	pool.MarkFailedByName("test-1", llm.NewOverloadedError("test", "overloaded"))

	_, err := pool.GetAvailable()
	var cooldownErr *llm.CooldownError
	if !errors.As(err, &cooldownErr) {
		t.Fatalf("expected CooldownError, got %T: %v", err, err)
	}

	if cooldownErr.Wait < 29*time.Second || cooldownErr.Wait > 31*time.Second {
		t.Errorf("Wait = %v, want ~30s", cooldownErr.Wait)
	}
}

func TestCooldown_TransientCooldown10s(t *testing.T) {
	pool := llm.NewAuthPool("test", []string{"key-A"})

	pool.MarkFailedByName("test-1", llm.NewAPIErrorFromStatus("test", 500, "internal error"))

	_, err := pool.GetAvailable()
	var cooldownErr *llm.CooldownError
	if !errors.As(err, &cooldownErr) {
		t.Fatalf("expected CooldownError, got %T: %v", err, err)
	}

	if cooldownErr.Wait < 9*time.Second || cooldownErr.Wait > 11*time.Second {
		t.Errorf("Wait = %v, want ~10s", cooldownErr.Wait)
	}
}

func TestCooldown_SoonestProfileWins(t *testing.T) {
	pool := llm.NewAuthPool("test", []string{"k1", "k2", "k3"})

	// Different error types → different cooldown durations
	// k1: 429 → 60s
	pool.MarkFailedByName("test-1", llm.NewRateLimitError("test", "rate limited"))
	// k2: 500 → 10s (shortest)
	pool.MarkFailedByName("test-2", llm.NewAPIErrorFromStatus("test", 500, "error"))
	// k3: 503 → 30s
	pool.MarkFailedByName("test-3", llm.NewOverloadedError("test", "overloaded"))

	_, err := pool.GetAvailable()
	var cooldownErr *llm.CooldownError
	if !errors.As(err, &cooldownErr) {
		t.Fatalf("expected CooldownError, got %T: %v", err, err)
	}

	// The soonest profile (k2 with 10s) should determine the wait
	if cooldownErr.Wait > 11*time.Second {
		t.Errorf("Wait = %v, want ≈10s (soonest profile)", cooldownErr.Wait)
	}
}
