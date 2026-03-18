package stress_test

import (
	"errors"
	"fmt"
	"testing"

	llm "github.com/bds421/rho-llm"
)

func TestErrorChain_DeepWrapping(t *testing.T) {
	base := llm.NewRateLimitError("test", "rate limited")
	var err error = base
	for i := 0; i < 15; i++ {
		err = fmt.Errorf("layer %d: %w", i, err)
	}

	if !llm.IsRateLimited(err) {
		t.Error("IsRateLimited should be true through 15 layers of wrapping")
	}
	if !llm.IsRetryable(err) {
		t.Error("IsRetryable should be true through 15 layers of wrapping")
	}
}

func TestErrorChain_AllClassifiers(t *testing.T) {
	constructors := []struct {
		name string
		err  *llm.APIError
	}{
		{"RateLimit", llm.NewRateLimitError("test", "msg")},
		{"Overloaded", llm.NewOverloadedError("test", "msg")},
		{"Auth401", llm.NewAuthError("test", "msg", 401)},
		{"Auth403", llm.NewAuthError("test", "msg", 403)},
		{"ContextLength", llm.NewContextLengthError("test", "context length exceeded")},
	}

	classifiers := []struct {
		name string
		fn   func(error) bool
	}{
		{"IsRateLimited", llm.IsRateLimited},
		{"IsOverloaded", llm.IsOverloaded},
		{"IsAuthError", llm.IsAuthError},
		{"IsRetryable", llm.IsRetryable},
		{"IsContextLength", llm.IsContextLength},
	}

	expected := map[string]map[string]bool{
		"RateLimit":     {"IsRateLimited": true, "IsRetryable": true},
		"Overloaded":    {"IsOverloaded": true, "IsRetryable": true},
		"Auth401":       {"IsAuthError": true},
		"Auth403":       {"IsAuthError": true},
		"ContextLength": {"IsContextLength": true},
	}

	for _, c := range constructors {
		for wrapDepth := 0; wrapDepth <= 10; wrapDepth++ {
			var err error = c.err
			for i := 0; i < wrapDepth; i++ {
				err = fmt.Errorf("wrap %d: %w", i, err)
			}

			for _, cl := range classifiers {
				got := cl.fn(err)
				want := expected[c.name][cl.name]
				if got != want {
					t.Errorf("%s wrapped %d times: %s() = %v, want %v",
						c.name, wrapDepth, cl.name, got, want)
				}
			}
		}
	}
}

func TestErrorChain_CooldownErrorUnwrap(t *testing.T) {
	var err error = &llm.CooldownError{Wait: 30}
	for i := 0; i < 5; i++ {
		err = fmt.Errorf("layer %d: %w", i, err)
	}

	if !errors.Is(err, llm.ErrNoAvailableProfiles) {
		t.Error("errors.Is(ErrNoAvailableProfiles) should be true through 5 layers")
	}

	var cooldownErr *llm.CooldownError
	if !errors.As(err, &cooldownErr) {
		t.Error("errors.As(*CooldownError) should work through 5 layers")
	}
}

func TestErrorChain_NewAPIErrorFromStatus_AllCodes(t *testing.T) {
	tests := []struct {
		status    int
		body      string
		wantRate  bool
		wantOver  bool
		wantAuth  bool
		wantCtx   bool
		wantRetry bool
	}{
		{429, "rate limited", true, false, false, false, true},
		{503, "overloaded", false, true, false, false, true},
		{401, "unauthorized", false, false, true, false, false},
		{403, "forbidden", false, false, true, false, false},
		{400, "context length exceeded", false, false, false, true, false},
		{400, "bad request", false, false, false, false, false},
		{500, "internal server error", false, false, false, false, true},
		{502, "bad gateway", false, false, false, false, true},
		{408, "request timeout", false, false, false, false, true},
		{404, "not found", false, false, false, false, false},
	}

	for _, tt := range tests {
		err := llm.NewAPIErrorFromStatus("test", tt.status, tt.body)
		name := fmt.Sprintf("status_%d_%s", tt.status, tt.body)

		if got := llm.IsRateLimited(err); got != tt.wantRate {
			t.Errorf("%s: IsRateLimited = %v, want %v", name, got, tt.wantRate)
		}
		if got := llm.IsOverloaded(err); got != tt.wantOver {
			t.Errorf("%s: IsOverloaded = %v, want %v", name, got, tt.wantOver)
		}
		if got := llm.IsAuthError(err); got != tt.wantAuth {
			t.Errorf("%s: IsAuthError = %v, want %v", name, got, tt.wantAuth)
		}
		if got := llm.IsContextLength(err); got != tt.wantCtx {
			t.Errorf("%s: IsContextLength = %v, want %v", name, got, tt.wantCtx)
		}
		if got := llm.IsRetryable(err); got != tt.wantRetry {
			t.Errorf("%s: IsRetryable = %v, want %v", name, got, tt.wantRetry)
		}
	}
}

func TestIsRetryable_NonAPIErrors(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"connection refused", true},
		{"no such host", true},
		{"request timeout", true},
		{"unexpected eof", true},
		{"connection reset by peer", true},
		{"broken pipe", true},
		// "request failed: dial tcp" — in production, dial errors are wrapped
		// net.Error types caught by the typed check in IsRetryable, not by
		// string matching. The overly broad "request failed" pattern was removed
		// to prevent false positives on non-retryable errors.
		{"request failed: dial tcp", false},
		{"permission denied", false},
		{"invalid json", false},
		{"", false},
	}

	for _, tt := range tests {
		err := errors.New(tt.msg)
		if got := llm.IsRetryable(err); got != tt.want {
			t.Errorf("IsRetryable(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}

	// nil error
	if llm.IsRetryable(nil) {
		t.Error("IsRetryable(nil) should be false")
	}
}
