package main

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/bds421/rho-llm"
)

func TestPresets(t *testing.T) {
	sonar, ok := llm.PresetFor("perplexity")
	if !ok || sonar.BaseURL != "https://api.perplexity.ai" || sonar.Protocol != "openai_compat" || sonar.AuthHeader != "Bearer" {
		t.Errorf("perplexity = %+v", sonar)
	}
	agent, ok := llm.PresetFor("perplexity-agent")
	if !ok || agent.BaseURL != "https://api.perplexity.ai/v1" || agent.Protocol != "openai_responses" || agent.AuthHeader != "Bearer" {
		t.Errorf("perplexity-agent = %+v", agent)
	}
	if got := llm.GetDefaultModel("perplexity-agent"); got != "" {
		t.Errorf("agent default model = %q, want none (opt-in, pick explicitly)", got)
	}
}

func TestSonarCatalog(t *testing.T) {
	want := []string{"sonar", "sonar-pro", "sonar-reasoning-pro", "sonar-deep-research"}
	have := map[string]bool{}
	for _, m := range llm.ModelsByProvider("perplexity") {
		have[m.ID] = true
		if strings.Contains(m.ID, "/") {
			t.Errorf("Agent-style ID %q in the Sonar catalog", m.ID)
		}
	}
	for _, id := range want {
		if !have[id] {
			t.Errorf("%s missing from the Sonar catalog", id)
		}
		if _, retired := llm.RetiredModelReplacement(id); retired {
			t.Errorf("%s is marked retired", id)
		}
	}
	// No global "sonar" alias: it would silently turn the Sonar model into the
	// Agent model of the same name.
	if got := llm.ResolveModelAlias("sonar"); got != "sonar" {
		t.Errorf("sonar alias = %q", got)
	}
}

func TestAgentCatalog(t *testing.T) {
	agent := llm.ModelsByProvider("perplexity-agent")
	if len(agent) < 10 {
		t.Fatalf("only %d rows", len(agent))
	}
	vendors := map[string]bool{}
	for _, m := range agent {
		v, _, found := strings.Cut(m.ID, "/")
		if !found {
			t.Errorf("%q has no vendor prefix", m.ID)
		}
		vendors[v] = true
		if m.InputPricePer1M <= 0 || m.OutputPricePer1M <= 0 {
			t.Errorf("%s unpriced", m.ID)
		}
		if m.CacheReadPricePer1M <= 0 || m.CacheReadPricePer1M >= m.InputPricePer1M {
			t.Errorf("%s cache read price %v vs input %v", m.ID, m.CacheReadPricePer1M, m.InputPricePer1M)
		}
	}
	for _, v := range []string{"perplexity", "anthropic", "google"} {
		if !vendors[v] {
			t.Errorf("no %s models in the Agent catalog", v)
		}
	}
}

func TestCrossCatalogRefusalCapabilityLevel(t *testing.T) {
	for _, m := range llm.ModelsByProvider("perplexity-agent") {
		if _, err := llm.ResolveCapabilityProfile(llm.Config{Provider: "perplexity", Model: m.ID}); err == nil {
			t.Errorf("Agent model %s accepted on the Sonar preset", m.ID)
		}
	}
	for _, m := range llm.ModelsByProvider("perplexity") {
		if _, err := llm.ResolveCapabilityProfile(llm.Config{Provider: "perplexity-agent", Model: m.ID}); err == nil {
			t.Errorf("Sonar model %s accepted on the Agent preset", m.ID)
		}
	}
	// Hostile model strings: never resolve, never panic.
	for _, id := range []string{"", " ", "perplexity/", "/sonar", "PERPLEXITY/SONAR", "perplexity/sonar\x00", "perplexity/sonar ", "../sonar"} {
		if _, err := llm.ResolveCapabilityProfile(llm.Config{Provider: "perplexity-agent", Model: id}); err == nil {
			t.Errorf("model %q resolved", id)
		}
	}
}

// The refusal must happen before any request: zero HTTP hits, on Complete and
// on Stream, whichever stage (NewClient or the call) raises it.
func TestCrossCatalogRefusalMakesZeroHTTPCalls(t *testing.T) {
	f := newFake(t, "application/json", sonarJSON)
	for _, tc := range []struct{ provider, model string }{
		{"perplexity", "perplexity/sonar"},
		{"perplexity", "anthropic/claude-sonnet-5-5"},
		{"perplexity-agent", "sonar-pro"},
		{"perplexity-agent", "sonar"},
	} {
		c, err := llm.NewClient(llm.Config{Provider: tc.provider, Model: tc.model, APIKey: "k", BaseURL: f.URL, MaxTokens: 10, MaxRetries: 1})
		if err != nil {
			continue // refused at construction: no traffic possible
		}
		if _, err := c.Complete(context.Background(), userReq("x")); err == nil {
			t.Errorf("%s/%s: Complete succeeded", tc.provider, tc.model)
		}
		for _, err := range c.Stream(context.Background(), userReq("x")) {
			if err == nil {
				continue
			}
		}
		c.Close()
	}
	if n := f.hits.Load(); n != 0 {
		t.Fatalf("cross-catalog use reached the network %d times", n)
	}
}

func TestCrossCatalogRefusalConcurrent(t *testing.T) {
	f := newFake(t, "application/json", sonarJSON)
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := llm.NewClient(llm.Config{Provider: "perplexity", Model: "perplexity/sonar", APIKey: "k", BaseURL: f.URL, MaxTokens: 10, MaxRetries: 1})
			if err != nil {
				return
			}
			defer c.Close()
			_, _ = c.Complete(context.Background(), userReq("x"))
		}()
	}
	wg.Wait()
	if f.hits.Load() != 0 {
		t.Fatal("concurrent refusal leaked requests")
	}
}
