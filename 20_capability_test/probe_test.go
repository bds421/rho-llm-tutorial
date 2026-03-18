package capability_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

// TestProbeHardQuestions sends candidate Level-6 questions to sonnet-4-6
// to find ones it CANNOT solve. Run with:
//
//	go test -v -run TestProbeHardQuestions -timeout 10m .
func TestProbeHardQuestions(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	cfg := llm.Config{
		Provider:  "anthropic",
		Model:     "claude-sonnet-4-6",
		APIKey:    apiKey,
		MaxTokens: 2048,
		Timeout:   60 * time.Second,
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	probes := []struct {
		id       string
		prompt   string
		expected string // what correct answer looks like
	}{
		{
			// Self-referential: d0 counts 0s, d1 counts 1s, ..., d9 counts 9s.
			// Unique solution: 6210001000.
			// d0=6 (six 0s), d1=2 (two 1s), d2=1 (one 2), d3=0, d4=0, d5=0,
			// d6=1 (one 6), d7=0, d8=0, d9=0.
			// Verify: digits are 6,2,1,0,0,0,1,0,0,0. Count of 0s=6✓, 1s=2✓, 2s=1✓,
			// 3s=0✓, 4s=0✓, 5s=0✓, 6s=1✓, 7s=0✓, 8s=0✓, 9s=0✓.
			id: "self-referential-sequence",
			prompt: "Find a 10-digit sequence d0 d1 d2 d3 d4 d5 d6 d7 d8 d9 (leading zeros allowed) " +
				"where digit di equals the number of times digit i appears in the entire sequence. " +
				"For example, d3 tells you how many 3s exist across all ten positions. " +
				"Reply with just the 10 digits in order.",
			expected: "6210001000",
		},
		{
			// A sees B=2, C=3. A's number is either 1 (=3-2) or 5 (=2+3).
			// B sees A=1, C=3. B's number is either 2 (=3-1) or 4 (=1+3).
			// C sees A=1, B=2. C's number is either 1 (=2-1) or 3 (=1+2).
			// But C≠1 because C must be a positive integer AND all are told exactly one
			// equals the sum of the other two. If C=1: numbers are 1,2,1 → 2=1+1 ✓ but
			// also need "exactly one" constraint. Actually the constraint is just that
			// one of the three equals the sum of the other two.
			// A says "No": A has two possibilities (1 or 5). Can't eliminate either.
			// But wait — if A=5, then 5=2+3 ✓. If A=1, then 3=1+2 ✓. Both valid. So A says No.
			// B says "No": B could be 2 or 4. If B=2: 3=1+2 ✓. If B=4: 4=1+3 ✓. Both valid. B says No.
			// C thinks: C could be 1 or 3.
			// If C=1: numbers are (1,2,1). Sum relation: 2=1+1 ✓.
			//   But then A would see B=2,C=1. A's options: |2-1|=1 or 2+1=3. A=1 or A=3.
			//   If A=1: (1,2,1) → 2=1+1 ✓. If A=3: (3,2,1) → 3=2+1 ✓. Both valid → A says No. Consistent.
			//   B would see A=1,C=1. B's options: |1-1|=0 (not positive!) or 1+1=2. B must be 2. B would KNOW!
			//   But B said "No". Contradiction! So C≠1.
			// Therefore C=3.
			id: "epistemic-logic",
			prompt: "Three perfect logicians (A, B, C) each have a positive integer on their forehead. " +
				"They see the other two but not their own. They're told: 'Exactly one of the three " +
				"numbers equals the sum of the other two.' " +
				"A sees B=2, C=3. B sees A=1, C=3. C sees A=1, B=2. " +
				"Asked in turn, A says 'No, I don't know my number.' " +
				"B says 'No.' C says 'Yes, I know my number.' " +
				"What is C's number? Reply with just the number.",
			expected: "3 — if C=1 then B would see (1,1) and know B=2; B said No, so C≠1, thus C=3",
		},
		{
			// Hour hand: 360°/12h = 0.5°/min. Minute hand: 6°/min. Second hand: 360°/min.
			// At time t (minutes from 12:00):
			//   H = 0.5t mod 360, M = 6t mod 360, S = 360t mod 360.
			// For 120° trisection: {H,M,S} must be {x, x+120, x+240} mod 360 for some x.
			// This requires solving three simultaneous modular equations.
			// The second hand moves so fast (6°/sec) relative to H and M that the
			// system of equations has no exact solution. The key insight:
			// M-H = 5.5t mod 360. S-H = 359.5t mod 360. S-M = 354t mod 360.
			// Need two of {M-H, S-H, S-M} = ±120 mod 360.
			// M-H = 120: 5.5t = 120+360k → t = (120+360k)/5.5
			// S-M = 120: 354t = 120+360j → t = (120+360j)/354
			// Setting equal: (120+360k)/5.5 = (120+360j)/354
			// 354(120+360k) = 5.5(120+360j)
			// 42480 + 127440k = 660 + 1980j
			// 41820 + 127440k = 1980j
			// 41820/1980 + 127440k/1980 = j → 21.12... not integer for k=0.
			// 127440/1980 = 64.36... irrational ratio → no integer solutions exist.
			// Answer: 0 times.
			id: "clock-trisection",
			prompt: "On a standard analog clock, the hour hand, minute hand, and second hand all move " +
				"continuously at their usual constant rates. How many times in a 12-hour period are " +
				"all three hands exactly 120 degrees apart from each other (i.e., perfectly trisecting " +
				"the clock face)? Reply with just the number.",
			expected: "0 — the angular velocities are incommensurate; the system of equations has no solution",
		},
	}

	ctx := context.Background()
	for _, p := range probes {
		t.Run(p.id, func(t *testing.T) {
			resp, err := client.Complete(ctx, llm.Request{
				Messages: []llm.Message{
					llm.NewTextMessage(llm.RoleUser, p.prompt),
				},
				MaxTokens: 2048,
			})
			if err != nil {
				t.Fatalf("API error: %v", err)
			}
			fmt.Fprintf(os.Stderr, "\n=== %s ===\nExpected: %s\nModel response:\n%s\n\n", p.id, p.expected, resp.Content)
			t.Logf("\n=== %s ===\nExpected: %s\nResponse:\n%s", p.id, p.expected, resp.Content)
		})
	}
}
