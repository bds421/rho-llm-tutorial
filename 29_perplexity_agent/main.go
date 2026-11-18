// Tutorial 29: Perplexity Presets (Sonar and the Agent API)
//
// Demonstrates: the "perplexity" preset (Sonar on /chat/completions) next to the
//               opt-in "perplexity-agent" preset (Responses wire format with a
//               Perplexity dialect), the two model catalogs, cache-read cost
//               estimation, the cross-catalog refusal, and Agent API
//               passthrough (effort / max_steps via SamplingParams).
//
// Works offline; with PERPLEXITY_API_KEY set it also makes one real Sonar call
// and one real Agent call. The Agent API wire shapes come from Perplexity's
// documentation and were not verified live by this tutorial.

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

func main() {
	// === Step 1: the two presets ===
	fmt.Println("=== Step 1: Presets ===")
	for _, name := range []string{"perplexity", "perplexity-agent"} {
		p, ok := llm.PresetFor(name)
		fmt.Printf("  %-17s found=%v BaseURL=%-32s Protocol=%s\n", name, ok, p.BaseURL, p.Protocol)
	}

	// === Step 2: the two catalogs ===
	fmt.Println("\n=== Step 2: Catalogs ===")
	for _, name := range []string{"perplexity", "perplexity-agent"} {
		models := llm.ModelsByProvider(name)
		fmt.Printf("  %s: %d models\n", name, len(models))
		for _, m := range models {
			fmt.Printf("    %-42s in $%.3f  out $%.2f  cache-read $%.4f /1M\n",
				m.ID, m.InputPricePer1M, m.OutputPricePer1M, m.CacheReadPricePer1M)
		}
	}

	// === Step 3: cost with cache reads ===
	// Usage is disjoint: InputTokens excludes the cached share.
	fmt.Println("\n=== Step 3: Cost estimate with cache reads ===")
	const agentModel = "anthropic/claude-sonnet-5-5"
	cold := llm.EstimateCost(llm.CostInput{Model: agentModel, InputTokens: 1_000_000, OutputTokens: 100_000})
	warm := llm.EstimateCost(llm.CostInput{Model: agentModel, InputTokens: 600_000, CacheReadTokens: 400_000, OutputTokens: 100_000})
	fmt.Printf("  %s, 1M in + 100k out: cold $%.4f, 40%% cached $%.4f\n", agentModel, cold, warm)

	// === Step 4: a model from one catalog is refused on the other preset ===
	fmt.Println("\n=== Step 4: Cross-catalog refusal ===")
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "should never be reached", http.StatusTeapot)
	}))
	defer srv.Close()
	for _, c := range []struct{ provider, model string }{
		{"perplexity", "perplexity/sonar"}, // Agent ID on the Sonar preset
		{"perplexity-agent", "sonar-pro"},  // Sonar ID on the Agent preset
		{"perplexity-agent", agentModel},   // control: allowed
	} {
		_, err := llm.ResolveCapabilityProfile(llm.Config{Provider: c.provider, Model: c.model})
		fmt.Printf("  %-17s %-30s -> %v\n", c.provider, c.model, errOrOK(err))
	}
	cl, err := llm.NewClient(llm.Config{Provider: "perplexity", Model: "perplexity/sonar", APIKey: "offline", BaseURL: srv.URL, MaxTokens: 16})
	if err == nil {
		_, err = cl.Complete(context.Background(), llm.Request{Messages: []llm.Message{llm.NewTextMessage(llm.RoleUser, "hi")}})
		cl.Close()
	}
	fmt.Printf("  Complete via wrong catalog: err=%v, HTTP hits=%d\n", err != nil, hits.Load())

	// === Step 5: live calls (only with a key) ===
	fmt.Println("\n=== Step 5: Live calls ===")
	key := os.Getenv("PERPLEXITY_API_KEY")
	if key == "" {
		fmt.Println("  PERPLEXITY_API_KEY not set: skipping the real Sonar and Agent calls.")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	ask := func(cfg llm.Config, req llm.Request) {
		cfg.APIKey = key
		c, err := llm.NewClient(cfg)
		if err != nil {
			fmt.Printf("  %s: client error: %v\n", cfg.Provider, err)
			return
		}
		defer c.Close()
		req.Messages = []llm.Message{llm.NewTextMessage(llm.RoleUser, "In one sentence: what is the Go programming language?")}
		resp, err := c.Complete(ctx, req)
		if err != nil {
			fmt.Printf("  %s: error: %v\n", cfg.Provider, err)
			return
		}
		fmt.Printf("  %s (%s): %q (in=%d cached=%d out=%d)\n", cfg.Provider, cfg.Model, resp.Content, resp.InputTokens, resp.CacheReadTokens, resp.OutputTokens)
	}
	ask(llm.Config{Provider: "perplexity", Model: "sonar", MaxTokens: 200}, llm.Request{})
	ask(llm.Config{Provider: "perplexity-agent", Model: "perplexity/sonar", MaxTokens: 400},
		llm.Request{System: "Answer in one sentence.", SamplingParams: map[string]any{"max_steps": 1}})
}

func errOrOK(err error) string {
	if err == nil {
		return "OK"
	}
	return "refused: " + err.Error()
}
