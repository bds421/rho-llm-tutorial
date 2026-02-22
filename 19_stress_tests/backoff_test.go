package stress_test

import (
	"testing"
	"time"

	llm "gitlab2024.bds421-cloud.com/bds421/rho/llm"
)

func TestBackoff_ExponentialGrowth(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	const samples = 1000

	for attempt := 0; attempt <= 10; attempt++ {
		// Expected center: min(baseDelay * 2^attempt, maxDelay)
		center := baseDelay
		for i := 0; i < attempt; i++ {
			center *= 2
			if center > maxDelay {
				center = maxDelay
				break
			}
		}
		if center > maxDelay {
			center = maxDelay
		}

		lo := time.Duration(float64(center) * 0.75)
		hi := maxDelay // jitter can't exceed maxDelay

		for s := 0; s < samples; s++ {
			d := llm.Backoff(attempt, baseDelay, maxDelay)
			if d < lo || d > hi {
				t.Errorf("attempt %d sample %d: got %v, want [%v, %v]", attempt, s, d, lo, hi)
			}
		}
	}
}

func TestBackoff_JitterDistribution(t *testing.T) {
	const samples = 10000
	baseDelay := 4 * time.Second
	maxDelay := 30 * time.Second

	// Test attempt 0 where center = 4s, range = [3s, 5s]
	// Divide into 4 quartiles
	quartiles := [4]int{}
	lo := float64(3 * time.Second)
	hi := float64(5 * time.Second)
	qWidth := (hi - lo) / 4

	for i := 0; i < samples; i++ {
		d := float64(llm.Backoff(0, baseDelay, maxDelay))
		q := int((d - lo) / qWidth)
		if q < 0 {
			q = 0
		}
		if q > 3 {
			q = 3
		}
		quartiles[q]++
	}

	for i, count := range quartiles {
		pct := float64(count) / float64(samples)
		if pct < 0.15 {
			t.Errorf("quartile %d has only %.1f%% of samples (want >= 15%%)", i, pct*100)
		}
	}
}

func TestBackoff_CappedAtMaxDelay(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 5 * time.Second

	for attempt := 0; attempt <= 20; attempt++ {
		for s := 0; s < 100; s++ {
			d := llm.Backoff(attempt, baseDelay, maxDelay)
			if d > maxDelay {
				t.Errorf("attempt %d: got %v > maxDelay %v", attempt, d, maxDelay)
			}
		}
	}
}

func TestBackoff_ZeroBaseDelay(t *testing.T) {
	// Should not panic
	d := llm.Backoff(0, 0, 10*time.Second)
	if d < 0 {
		t.Errorf("got negative duration: %v", d)
	}
}

func TestBackoff_BaseExceedsMax(t *testing.T) {
	maxDelay := 1 * time.Second
	for s := 0; s < 100; s++ {
		d := llm.Backoff(0, 10*time.Second, maxDelay)
		if d > maxDelay {
			t.Errorf("got %v > maxDelay %v", d, maxDelay)
		}
	}
}

func BenchmarkBackoff(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		llm.Backoff(i%10, time.Second, 30*time.Second)
	}
}
