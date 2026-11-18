// Tutorial 28: Gemini Frontier — Gemini 4, long decode, the Interactions API
//
// Demonstrates: gemini-4-carbon / gemini-4-hydrogen registry rows, EstimateCost,
//               per-model thinking levels (xhigh only on Hydrogen, "minimal" sent
//               as "low" where Google answers 400), Request.CachedContent refusal
//               on Gemini 4, Request.SamplingParams sampling-key refusal vs
//               Request.Temperature, the opt-in "gemini-interactions" provider
//               (ResolveProtocol, Complete + Stream, store:false by default),
//               Response.RawStopReason, Config.HTTPClient (to watch the wire).
//
// Long decode (a Gemini 4 turn that ends with finishReason CONTINUATION and is
// resumed with a continuationToken) is transparent to the caller and only
// reproducible with a server that answers that way, so it is exercised by the
// offline tests (go test -short -race ./...), not by this program.
//
// Steps 1-3 need no key and never touch the network. Steps 4-5 call the Gemini
// API (GEMINI_API_KEY) with tiny prompts; without a key they are skipped.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

// deadBase is where the offline steps point the client. Every refusal in
// steps 1-3 happens before a request is built, so nothing is ever sent; if one
// were, it would fail to connect instead of reaching Google.
const deadBase = "http://127.0.0.1:9"

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	// =========================================================================
	// Step 1: The Gemini 4 rows in the registry, and what a turn costs
	// =========================================================================
	fmt.Println("=== Step 1: registry rows and cost ===")
	for _, id := range []string{"gemini-4-carbon", "gemini-4-hydrogen", "gemini-3.8-flash"} {
		info, ok := llm.GetModelInfo(id)
		if !ok {
			fmt.Printf("  %-18s not in the registry\n", id)
			continue
		}
		fmt.Printf("  %-18s per-request output limit %6d | context %7d | $%.3f in / $%.2f out / $%.3f cache-read per 1M\n",
			id, info.MaxTokens, info.ContextWindow, info.InputPricePer1M, info.OutputPricePer1M, info.CacheReadPricePer1M)
	}
	// MaxTokens is ONE request's limit. On Gemini 4 a turn can run past it (long
	// decode) up to the shared context window, so a bigger Request.MaxTokens is
	// not clamped to it.
	turn := llm.CostInput{Model: "gemini-4-hydrogen", InputTokens: 20_000, CacheReadTokens: 80_000, OutputTokens: 40_000, ThinkingTokens: 15_000}
	fmt.Printf("  a long-decode turn on Hydrogen (20k in, 80k cached, 40k out, 15k thinking) ~ $%.4f\n\n", llm.EstimateCost(turn))

	// =========================================================================
	// Step 2: Which wire protocol does a provider name select?
	// =========================================================================
	fmt.Println("=== Step 2: presets ===")
	for _, name := range []string{"gemini", "google", "gemini-interactions"} {
		cfg := llm.Config{Provider: name}
		fmt.Printf("  %-20s protocol %-20s base %s\n", name, llm.ResolveProtocol(cfg), llm.ResolveBaseURL(cfg))
	}
	fmt.Println("  (gemini/google stay on generateContent; gemini-interactions is opt-in)")
	fmt.Println()

	// =========================================================================
	// Step 3: Refusals that happen before any request is sent
	// =========================================================================
	fmt.Println("=== Step 3: refusals (offline) ===")
	offline := func(model string) llm.Client {
		c, err := llm.NewClient(llm.Config{Provider: "gemini", Model: model, APIKey: "unused", BaseURL: deadBase, DisableRetries: true})
		if err != nil {
			fmt.Fprintf(os.Stderr, "NewClient(%s): %v\n", model, err)
			os.Exit(1)
		}
		return c
	}
	hello := []llm.Message{llm.NewTextMessage(llm.RoleUser, "hi")}
	show := func(label string, err error) {
		if err == nil {
			fmt.Printf("  %-52s NOT refused (unexpected)\n", label)
			return
		}
		msg := err.Error()
		if len(msg) > 110 {
			msg = msg[:110] + "..."
		}
		fmt.Printf("  %-52s refused: %s\n", label, msg)
	}

	carbon := offline("gemini-4-carbon")
	_, err := carbon.Complete(ctx, llm.Request{Messages: hello, CachedContent: "cachedContents/abc"})
	show("gemini-4-carbon + CachedContent", err)

	// SamplingParams is merged at the TOP level of the body; Gemini keeps
	// sampling inside generationConfig, so a top-level key is a 400 "Unknown name".
	for _, key := range []string{"temperature", "topK"} {
		_, err = offline("gemini-2.5-flash").Complete(ctx, llm.Request{Messages: hello, SamplingParams: map[string]any{key: 0.5}})
		show(fmt.Sprintf("gemini-2.5-flash + SamplingParams[%q]", key), err)
	}

	// The typed field is the way. gemini-2.5-flash takes it (it goes into
	// generationConfig); the models that dropped sampling controls refuse it
	// up front instead of sending a value Google ignores.
	temp := 0.2
	_, err = offline("gemini-3.8-flash").Complete(ctx, llm.Request{Messages: hello, Temperature: &temp})
	show("gemini-3.8-flash + Request.Temperature", err)
	_, err = offline("gemini-4-hydrogen").Complete(ctx, llm.Request{Messages: hello, Temperature: &temp})
	show("gemini-4-hydrogen + Request.Temperature", err)
	fmt.Println()

	// =========================================================================
	// Steps 4-5 call the API
	// =========================================================================
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		fmt.Println("GEMINI_API_KEY is not set: skipping the live steps (4 and 5).")
		fmt.Println("Set it (source .env) to see thinking levels and the Interactions API against Google.")
		return
	}

	// =========================================================================
	// Step 4: Thinking levels per model, watched on the wire
	// =========================================================================
	fmt.Println("=== Step 4: thinking levels (live) ===")
	fmt.Println("  Request.ThinkingLevel -> thinkingLevel the adapter sent -> HTTP status")
	spy := &spyTransport{}
	httpClient := &http.Client{Transport: spy, Timeout: 90 * time.Second}
	live := func(provider, model string) llm.Client {
		c, err := llm.NewClient(llm.Config{Provider: provider, Model: model, APIKey: key, HTTPClient: httpClient, DisableRetries: true})
		if err != nil {
			fmt.Fprintf(os.Stderr, "NewClient(%s): %v\n", model, err)
			os.Exit(1)
		}
		return c
	}
	plan := []struct {
		model string
		level llm.ThinkingLevel
	}{
		{"gemini-3.8-flash", llm.ThinkingMinimal}, // 400 on this model: sent as "low"
		{"gemini-3.8-flash", llm.ThinkingXHigh},   // xhigh is Hydrogen-only: sent as "high"
		{"gemini-4-hydrogen", llm.ThinkingXHigh},  // the one model with a real "xhigh"
		{"gemini-4-hydrogen", llm.ThinkingMinimal},
	}
	for _, p := range plan {
		resp, err := live("gemini", p.model).Complete(ctx, llm.Request{
			Messages:      []llm.Message{llm.NewTextMessage(llm.RoleUser, "Reply with the single word ok.")},
			MaxTokens:     200,
			ThinkingLevel: p.level,
		})
		status := "ok"
		if err != nil {
			status = "error: " + firstLine(err.Error())
		}
		fmt.Printf("  %-18s %-8s -> %-6s -> HTTP %d  %s", p.model, p.level, spy.level, spy.status, status)
		if resp != nil {
			fmt.Printf("  (%q, stop %s/%s, thinking tokens %d)", strings.TrimSpace(resp.Content), resp.StopReason, resp.RawStopReason, resp.ThinkingTokens)
		}
		fmt.Println()
	}
	fmt.Println()

	// =========================================================================
	// Step 5: The Interactions API (opt-in), Complete and Stream
	// =========================================================================
	fmt.Println("=== Step 5: gemini-interactions (live) ===")
	ia := live("gemini-interactions", "gemini-3.8-flash")
	fmt.Printf("  Provider() = %q, protocol %s\n", ia.Provider(), llm.ResolveProtocol(llm.Config{Provider: "gemini-interactions"}))
	req := llm.Request{
		Messages:      []llm.Message{llm.NewTextMessage(llm.RoleUser, "What is 17*23? Answer with just the number.")},
		MaxTokens:     200,
		ThinkingLevel: llm.ThinkingLow,
	}
	resp, err := ia.Complete(ctx, req)
	if err != nil {
		fmt.Printf("  Complete: error: %s\n", firstLine(err.Error()))
	} else {
		fmt.Printf("  Complete: %q stop %s (raw %q), in %d / out %d / thinking %d\n",
			strings.TrimSpace(resp.Content), resp.StopReason, resp.RawStopReason, resp.InputTokens, resp.OutputTokens, resp.ThinkingTokens)
		fmt.Printf("            sent to %s, store=%s\n", spy.path, spy.store)
	}
	fmt.Print("  Stream:   ")
	for ev, err := range ia.Stream(ctx, req) {
		if err != nil {
			fmt.Printf("error: %s", firstLine(err.Error()))
			break
		}
		switch ev.Type {
		case llm.EventContent:
			fmt.Print(ev.Text)
		case llm.EventDone:
			fmt.Printf("  [done: %s, out %d tokens]", ev.StopReason, ev.OutputTokens)
		}
	}
	fmt.Println()
	fmt.Printf("            store=%s (stateless: Google keeps no copy unless SamplingParams[\"store\"]=true)\n", spy.store)
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	if len(s) > 160 {
		s = s[:160] + "..."
	}
	return s
}

// spyTransport records what the adapter put on the wire (never the key): the
// request path, the thinking level and the store flag, and the HTTP status.
type spyTransport struct {
	path   string
	level  string
	store  string
	status int
}

func (s *spyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	s.path, s.level, s.store, s.status = r.URL.Path, "-", "-", 0
	if r.Body != nil {
		raw, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(raw))
		var body map[string]any
		if json.Unmarshal(raw, &body) == nil {
			if v, ok := nested(body, "generationConfig", "thinkingConfig", "thinkingLevel").(string); ok {
				s.level = v
			}
			if v, ok := nested(body, "generation_config", "thinking_level").(string); ok {
				s.level = v
			}
			if v, ok := body["store"]; ok {
				s.store = fmt.Sprint(v)
			}
		}
	}
	resp, err := http.DefaultTransport.RoundTrip(r)
	if resp != nil {
		s.status = resp.StatusCode
	}
	return resp, err
}

func nested(m map[string]any, path ...string) any {
	var cur any = m
	for _, k := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	return cur
}
