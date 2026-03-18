package capability_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

// TestAnalyzeProblematicPrompts sends each problematic prompt to sonnet with
// an extra "think aloud" instruction to expose ambiguity and suggest a better prompt.
func TestAnalyzeProblematicPrompts(t *testing.T) {
	client, err := llm.NewClient(llm.Config{
		Provider:  "ollama",
		Model:     "deepseek-r1:14b",
		MaxTokens: 1024,
		Timeout:   300 * time.Second,
	})
	if err != nil {
		t.Fatalf("cannot create client: %v", err)
	}
	defer client.Close()

	type probe struct {
		name          string
		originalEN    string
		expectedEN    string
		notExpectedEN string
	}

	probes := []probe{
		{
			name:          "mensa-math",
			originalEN:    "What is the 4-digit number in which the first digit is one-fifth of the last, and the second and third digits are the last digit multiplied by 3? (Hint: The sum of all digits is 12). Reply with just the 4-digit number.",
			expectedEN:    "1155",
			notExpectedEN: "(none)",
		},
		{
			name:          "mensa-riddle",
			originalEN:    "A man is looking at a photograph. He says: 'Brothers and sisters, I have none. But that man's father is my father's son.' Who is in the photograph? Reply very concisely.",
			expectedEN:    "son",
			notExpectedEN: "himself / me",
		},
		{
			name:          "mensa-calendar",
			originalEN:    "The day before two days after the day before tomorrow is Saturday. What day is it today? Reply with just the day of the week.",
			expectedEN:    "Friday",
			notExpectedEN: "(none)",
		},
	}

	analysisPrompt := `You are evaluating a logic/IQ test question used to benchmark LLMs.

Question: %s

The test expects the answer: "%s"
The test rejects the answer: "%s"

Please do three things:
1. Solve the problem step by step and give your answer.
2. Identify if the problem wording is ambiguous, misleading or has multiple valid interpretations. Be specific about what causes confusion.
3. Suggest an improved version of the question that has exactly one unambiguous correct answer and is harder to misinterpret. Keep it concise.`

	for _, p := range probes {
		t.Run(p.name, func(t *testing.T) {
			prompt := fmt.Sprintf(analysisPrompt, p.originalEN, p.expectedEN, p.notExpectedEN)
			resp, err := complete(t.Context(), client, 60*time.Second, prompt)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			t.Logf("\n=== %s ===\n%s\n", p.name, resp)
		})
	}
}
