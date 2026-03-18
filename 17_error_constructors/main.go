// Tutorial 17: Error Constructors & Classification
//
// Demonstrates: NewRateLimitError, NewOverloadedError, NewAuthError,
//               NewContextLengthError, NewAPIErrorFromStatus, and round-trip
//               verification with IsRateLimited, IsOverloaded, IsAuthError,
//               IsContextLength, IsRetryable, errors.As
//
// No API keys required — all errors are constructed locally.

package main

import (
	"errors"
	"fmt"

	"github.com/bds421/rho-llm"
)

func main() {
	// =========================================================================
	// Step 1: Construct each error type and verify with Is* helpers
	// =========================================================================
	fmt.Println("=== Step 1: Named Error Constructors ===")

	cases := []struct {
		name string
		err  error
		checks map[string]func(error) bool
	}{
		{
			name: "NewRateLimitError",
			err:  llm.NewRateLimitError("anthropic", "rate limit exceeded"),
			checks: map[string]func(error) bool{
				"IsRateLimited": llm.IsRateLimited,
				"IsRetryable":   llm.IsRetryable,
			},
		},
		{
			name: "NewOverloadedError",
			err:  llm.NewOverloadedError("gemini", "service overloaded"),
			checks: map[string]func(error) bool{
				"IsOverloaded": llm.IsOverloaded,
				"IsRetryable":  llm.IsRetryable,
			},
		},
		{
			name: "NewAuthError (401)",
			err:  llm.NewAuthError("openai", "invalid api key", 401),
			checks: map[string]func(error) bool{
				"IsAuthError":  llm.IsAuthError,
				"IsRetryable":  llm.IsRetryable,
			},
		},
		{
			name: "NewAuthError (403)",
			err:  llm.NewAuthError("anthropic", "forbidden", 403),
			checks: map[string]func(error) bool{
				"IsAuthError":  llm.IsAuthError,
				"IsRetryable":  llm.IsRetryable,
			},
		},
		{
			name: "NewContextLengthError",
			err:  llm.NewContextLengthError("openai", "context length exceeded"),
			checks: map[string]func(error) bool{
				"IsContextLength": llm.IsContextLength,
				"IsRetryable":     llm.IsRetryable,
			},
		},
	}

	for _, tc := range cases {
		fmt.Printf("\n%s: %v\n", tc.name, tc.err)
		for name, fn := range tc.checks {
			fmt.Printf("  %s = %v\n", name, fn(tc.err))
		}

		// Verify errors.As round-trip
		var apiErr *llm.APIError
		if errors.As(tc.err, &apiErr) {
			fmt.Printf("  APIError: status=%d, provider=%s, retryable=%v\n",
				apiErr.StatusCode, apiErr.Provider, apiErr.Retryable)
		}
	}
	fmt.Println()

	// =========================================================================
	// Step 2: NewAPIErrorFromStatus — construct errors from HTTP status codes
	// =========================================================================
	fmt.Println("=== Step 2: NewAPIErrorFromStatus ===")

	statusCases := []struct {
		status int
		body   string
		checks []string
	}{
		{429, "rate limit exceeded", []string{"IsRateLimited", "IsRetryable"}},
		{503, "service unavailable", []string{"IsOverloaded", "IsRetryable"}},
		{401, "unauthorized", []string{"IsAuthError"}},
		{400, "context_length_exceeded: too many tokens", []string{"IsContextLength"}},
		{500, "internal server error", []string{"IsRetryable"}},
		{502, "bad gateway", []string{"IsRetryable"}},
		{400, "invalid request body", []string{}},
	}

	checkFuncs := map[string]func(error) bool{
		"IsRateLimited":   llm.IsRateLimited,
		"IsOverloaded":    llm.IsOverloaded,
		"IsAuthError":     llm.IsAuthError,
		"IsContextLength": llm.IsContextLength,
		"IsRetryable":     llm.IsRetryable,
	}

	for _, sc := range statusCases {
		err := llm.NewAPIErrorFromStatus("test-provider", sc.status, sc.body)
		fmt.Printf("\nStatus %d (%s): %v\n", sc.status, sc.body, err)

		for _, checkName := range sc.checks {
			fn := checkFuncs[checkName]
			fmt.Printf("  %s = %v\n", checkName, fn(err))
		}

		// Verify errors.As works
		var apiErr *llm.APIError
		if errors.As(err, &apiErr) {
			fmt.Printf("  APIError: status=%d, retryable=%v\n", apiErr.StatusCode, apiErr.Retryable)
		}
	}
	fmt.Println()

	// =========================================================================
	// Step 3: errors.As round-trip through fmt.Errorf wrapping
	// =========================================================================
	fmt.Println("=== Step 3: Error Wrapping Round-Trip ===")

	original := llm.NewRateLimitError("anthropic", "too many requests")
	wrapped := fmt.Errorf("llm call failed: %w", original)
	doubleWrapped := fmt.Errorf("handler error: %w", wrapped)

	fmt.Printf("Original:      IsRateLimited=%v\n", llm.IsRateLimited(original))
	fmt.Printf("Wrapped:       IsRateLimited=%v\n", llm.IsRateLimited(wrapped))
	fmt.Printf("DoubleWrapped: IsRateLimited=%v\n", llm.IsRateLimited(doubleWrapped))

	var extracted *llm.APIError
	if errors.As(doubleWrapped, &extracted) {
		fmt.Printf("Extracted from double-wrap: status=%d, provider=%s\n",
			extracted.StatusCode, extracted.Provider)
	}
}
