package stress_test

import (
	"context"
	"sync"
	"testing"
	"time"

	llm "gitlab2024.bds421-cloud.com/bds421/rho/llm"
)

func TestAuthPool_ConcurrentGetAvailable(t *testing.T) {
	keys := make([]string, 5)
	for i := range keys {
		keys[i] = "key-" + string(rune('A'+i))
	}
	pool := llm.NewAuthPool("test", keys)

	var wg sync.WaitGroup
	const goroutines = 50
	const callsPerGoroutine = 100

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < callsPerGoroutine; i++ {
				profile, err := pool.GetAvailable()
				if err != nil {
					t.Errorf("GetAvailable returned error: %v", err)
					return
				}
				if profile.Name == "" {
					t.Error("GetAvailable returned empty profile name")
					return
				}
				if profile.APIKey == "" {
					t.Error("GetAvailable returned empty API key")
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestAuthPool_ConcurrentMarkFailedAndGetAvailable(t *testing.T) {
	keys := []string{"key-A", "key-B", "key-C"}
	pool := llm.NewAuthPool("test", keys)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup

	// Readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				_, _ = pool.GetAvailable()
			}
		}()
	}

	// Writers: mark failed then recover
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name := pool.Status() // just exercise Status under contention
			_ = name
			for ctx.Err() == nil {
				err := llm.NewOverloadedError("test", "overloaded")
				pool.MarkFailedByName("test-1", err)
				time.Sleep(time.Millisecond)
				pool.MarkSuccessByName("test-1")
			}
		}(i)
	}

	wg.Wait()

	if pool.Count() != 3 {
		t.Errorf("Count() = %d, want 3", pool.Count())
	}
}

func TestAuthPool_ConcurrentGetCurrentReadOnly(t *testing.T) {
	keys := []string{"key-A", "key-B", "key-C"}
	pool := llm.NewAuthPool("test", keys)

	var wg sync.WaitGroup
	const goroutines = 100

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				profile, ok := pool.GetCurrent()
				if !ok {
					t.Error("GetCurrent returned false")
					return
				}
				if profile.Name == "" {
					t.Error("GetCurrent returned empty name")
					return
				}
			}
		}()
	}
	wg.Wait()
}

func BenchmarkAuthPool_GetAvailable(b *testing.B) {
	keys := []string{"key-A", "key-B", "key-C"}
	pool := llm.NewAuthPool("bench", keys)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pool.GetAvailable()
	}
}

func BenchmarkAuthPool_GetAvailable_Parallel(b *testing.B) {
	keys := []string{"key-A", "key-B", "key-C"}
	pool := llm.NewAuthPool("bench", keys)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = pool.GetAvailable()
		}
	})
}
