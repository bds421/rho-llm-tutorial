package main

import (
	"strings"
	"testing"

	"github.com/bds421/rho-llm"
)

// Gemini 4 has no explicit caching: CachedContent is refused before the
// request (zero HTTP hits), for Complete and Stream. Gemini 3.8 Flash still
// takes it and sends cachedContent.
func TestCachedContentRefusedOnGemini4(t *testing.T) {
	for _, model := range []string{"gemini-4-carbon", "gemini-4-hydrogen", "gemini-4-argon"} {
		t.Run(model, func(t *testing.T) {
			f := newFake(t, jsonReply(okBody))
			c := geminiClient(t, f, model)
			req := userReq("hi")
			req.CachedContent = "cachedContents/abc123"

			_, err := c.Complete(t.Context(), req)
			if err == nil || !strings.Contains(err.Error(), "explicit caching") {
				t.Errorf("Complete err = %v, want the explicit-caching refusal", err)
			}
			r := drain(t, c, req)
			if r.Err == nil || !strings.Contains(r.Err.Error(), "explicit caching") || len(r.Events) != 0 {
				t.Errorf("Stream err = %v events %d, want the refusal and no events", r.Err, len(r.Events))
			}
			if f.hits() != 0 {
				t.Errorf("%d HTTP requests, want 0: the refusal must come before the wire", f.hits())
			}
		})
	}
}

func TestCachedContentStillWorksOnGemini38(t *testing.T) {
	f := newFake(t, jsonReply(okBody))
	c := geminiClient(t, f, "gemini-3.8-flash")
	req := userReq("hi")
	req.CachedContent = "cachedContents/abc123"
	if _, err := c.Complete(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	if f.req(0).Body["cachedContent"] != "cachedContents/abc123" {
		t.Errorf("cachedContent = %v", f.req(0).Body["cachedContent"])
	}
}

// SamplingParams is merged at the TOP level of the body; Gemini keeps sampling
// under generationConfig, so these keys are a 400 "Unknown name" live. The
// library refuses them on every Gemini model, before the request.
func TestSamplingParamsKeysRefused(t *testing.T) {
	models := []string{"gemini-2.5-flash", "gemini-3.8-flash", "gemini-4-carbon", "gemini-4-hydrogen"}
	keys := []string{"temperature", "top_p", "top_k", "topP", "topK"}
	for _, model := range models {
		for _, key := range keys {
			t.Run(model+"/"+key, func(t *testing.T) {
				f := newFake(t, jsonReply(okBody))
				c := geminiClient(t, f, model)
				req := userReq("hi")
				req.SamplingParams = map[string]any{key: 0.5}

				_, err := c.Complete(t.Context(), req)
				if err == nil || !strings.Contains(err.Error(), `"`+key+`"`) {
					t.Errorf("Complete err = %v, want a refusal naming %q", err, key)
				}
				if r := drain(t, c, req); r.Err == nil || len(r.Events) != 0 {
					t.Errorf("Stream err = %v events %d, want a refusal", r.Err, len(r.Events))
				}
				if f.hits() != 0 {
					t.Errorf("%d HTTP requests, want 0", f.hits())
				}
				if err != nil && strings.Contains(err.Error(), fakeKey) {
					t.Error("the refusal leaked the API key")
				}
			})
		}
	}
}

// Hostile: the refusal keys on the exact names, a nil or empty map is fine, and
// a refused key hidden among legitimate ones still stops the whole request.
func TestSamplingParamsHostileShapes(t *testing.T) {
	f := newFake(t, jsonReply(okBody))
	c := geminiClient(t, f, "gemini-2.5-flash")

	for name, params := range map[string]map[string]any{
		"nil":   nil,
		"empty": {},
	} {
		req := userReq("hi")
		req.SamplingParams = params
		if _, err := c.Complete(t.Context(), req); err != nil {
			t.Errorf("%s SamplingParams: %v", name, err)
		}
	}
	if f.hits() != 2 {
		t.Fatalf("hits = %d, want 2", f.hits())
	}

	req := userReq("hi")
	req.SamplingParams = map[string]any{"seed": 7, "topK": nil} // a nil value is still the key
	if _, err := c.Complete(t.Context(), req); err == nil {
		t.Error("topK: nil slipped through")
	}
	if f.hits() != 2 {
		t.Errorf("hits = %d after the refused request, want still 2", f.hits())
	}
}

// Request.Temperature is the typed way. It lands inside generationConfig on
// models that take it, and is refused up front (zero hits) on the models that
// dropped sampling controls, instead of being sent and ignored or 400ing.
func TestRequestTemperatureVersusSamplingParams(t *testing.T) {
	temp := 0.3

	t.Run("gemini-2.5-flash takes it", func(t *testing.T) {
		f := newFake(t, jsonReply(okBody))
		c := geminiClient(t, f, "gemini-2.5-flash")
		req := userReq("hi")
		req.Temperature = &temp
		if _, err := c.Complete(t.Context(), req); err != nil {
			t.Fatal(err)
		}
		if got := dig(f.req(0).Body, "generationConfig", "temperature"); got != 0.3 {
			t.Errorf("generationConfig.temperature = %v, want 0.3", got)
		}
		if _, top := f.req(0).Body["temperature"]; top {
			t.Error("temperature must never appear at the top level")
		}
	})

	for _, model := range []string{"gemini-3.8-flash", "gemini-4-carbon", "gemini-4-hydrogen"} {
		t.Run(model+" refuses it", func(t *testing.T) {
			f := newFake(t, jsonReply(okBody))
			c := geminiClient(t, f, model)
			req := userReq("hi")
			req.Temperature = &temp
			_, err := c.Complete(t.Context(), req)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), "temperature") {
				t.Errorf("err = %v, want a temperature refusal", err)
			}
			if f.hits() != 0 {
				t.Errorf("%d HTTP requests, want 0", f.hits())
			}
		})
	}
}

// Effort is not a Gemini knob (ThinkingLevel is).
func TestEffortRefusedOnGemini(t *testing.T) {
	f := newFake(t, jsonReply(okBody))
	c := geminiClient(t, f, "gemini-4-carbon")
	req := userReq("hi")
	req.Effort = "high"
	if _, err := c.Complete(t.Context(), req); err == nil || f.hits() != 0 {
		t.Errorf("err = %v hits %d, want a refusal and no request", err, f.hits())
	}
}

// The Gemini 4 rows exist in the registry with sane shape: per-request output
// limit below the context window (long decode continues past it).
func TestGemini4RegistryRows(t *testing.T) {
	for _, id := range []string{"gemini-4-carbon", "gemini-4-hydrogen", "gemini-4-argon", "gemini-4-neon"} {
		info, ok := llm.GetModelInfo(id)
		if !ok {
			t.Fatalf("%s missing from the registry", id)
		}
		if !info.SupportsThinking || info.ContextWindow != 1048576 || info.MaxTokens <= 0 || info.MaxTokens >= info.ContextWindow {
			t.Errorf("%s: %+v", id, info)
		}
		if info.InputPricePer1M <= 0 || info.OutputPricePer1M <= info.InputPricePer1M || info.CacheReadPricePer1M >= info.InputPricePer1M {
			t.Errorf("%s: implausible prices %+v", id, info)
		}
	}
	// Cost: 1M uncached input + 1M output + 1M cache reads on Hydrogen.
	got := llm.EstimateCost(llm.CostInput{Model: "gemini-4-hydrogen", InputTokens: 1_000_000, OutputTokens: 1_000_000, CacheReadTokens: 1_000_000})
	if want := 1.50 + 7.50 + 0.075; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("cost = %v, want %v", got, want)
	}
	// Neon (2026-11-18): $3 / $15, cache read $0.15.
	got = llm.EstimateCost(llm.CostInput{Model: "gemini-4-neon", InputTokens: 1_000_000, OutputTokens: 1_000_000, CacheReadTokens: 1_000_000})
	if want := 3.00 + 15.00 + 0.15; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("neon cost = %v, want %v", got, want)
	}
	// Hostile inputs never produce a negative or NaN estimate.
	if c := llm.EstimateCost(llm.CostInput{Model: "gemini-4-carbon", InputTokens: -5, OutputTokens: llm.TokensNotReported}); c != 0 {
		t.Errorf("negative counts cost %v, want 0", c)
	}
	if c := llm.EstimateCost(llm.CostInput{Model: "gemini-4-nope", InputTokens: 10}); c != 0 {
		t.Errorf("unknown model cost %v, want 0", c)
	}
}
