package main

import (
	"context"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/bds421/rho-llm"
)

const sonarJSON = `{"id":"c","model":"sonar","choices":[{"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`

func agentJSON(in, cached int) string {
	return `{"id":"resp_1","object":"response","status":"completed","model":"perplexity/sonar",` +
		`"output":[{"type":"message","content":[{"type":"output_text","text":"hi"}]}],` +
		`"usage":{"input_tokens":` + itoa(in) + `,"output_tokens":10,"total_tokens":` + itoa(in+10) +
		`,"input_tokens_details":{"cached_tokens":` + itoa(cached) + `}}}`
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestSonarRequestGoesToChatCompletions(t *testing.T) {
	f := newFake(t, "application/json", sonarJSON)
	c, err := llm.NewClient(llm.Config{Provider: "perplexity", Model: "sonar", APIKey: "test-key", BaseURL: f.URL, MaxTokens: 50, MaxRetries: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	req := userReq("hello")
	req.System = "be brief"
	if _, err := c.Complete(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	r := f.last(t)
	if r.Path != "/chat/completions" {
		t.Fatalf("path = %q", r.Path)
	}
	if r.Auth != "Bearer test-key" {
		t.Errorf("auth header = %q", r.Auth)
	}
	if _, ok := r.Body["messages"]; !ok {
		t.Errorf("Sonar body has no messages: %v", r.Body)
	}
	for _, k := range []string{"input", "instructions"} {
		if _, ok := r.Body[k]; ok {
			t.Errorf("Sonar body carries Responses field %q", k)
		}
	}
	if c.Provider() != "perplexity" {
		t.Errorf("Provider() = %q", c.Provider())
	}
}

func TestAgentRequestShape(t *testing.T) {
	f := newFake(t, "application/json", agentJSON(10, 0))
	c := agentClient(t, f.URL, "perplexity/sonar")
	req := userReq("hello")
	req.System = "be brief"
	req.SamplingParams = map[string]any{"max_steps": 3, "effort": "high"}
	if _, err := c.Complete(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	r := f.last(t)
	if r.Path != "/v1/responses" {
		t.Fatalf("path = %q, want /v1/responses", r.Path)
	}
	if r.Body["model"] != "perplexity/sonar" || r.Body["instructions"] != "be brief" {
		t.Errorf("body = %v", r.Body)
	}
	if r.Body["max_steps"] != float64(3) || r.Body["effort"] != "high" {
		t.Errorf("passthrough lost: %v", r.Body)
	}
	// The system prompt is the dialect's instructions, not an input item.
	if strings.Contains(r.Raw, `"role":"system"`) {
		t.Errorf("system prompt leaked into input: %s", r.Raw)
	}
	if c.Provider() != "perplexity-agent" {
		t.Errorf("Provider() = %q", c.Provider())
	}
}

func TestAgentMaxEffortKept(t *testing.T) {
	f := newFake(t, "application/json", agentJSON(10, 0))
	c := agentClient(t, f.URL, "perplexity/kimi-k3")
	req := userReq("hi")
	req.ThinkingLevel = llm.ThinkingMax
	if _, err := c.Complete(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	reasoning, _ := f.last(t).Body["reasoning"].(map[string]any)
	if reasoning["effort"] != "max" {
		t.Fatalf("reasoning = %v, want effort max (not downgraded to xhigh)", reasoning)
	}
}

// Hostile passthrough: SamplingParams may not overwrite adapter-owned fields.
func TestAgentSamplingParamsCannotOverrideAdapterFields(t *testing.T) {
	f := newFake(t, "application/json", agentJSON(10, 0))
	c := agentClient(t, f.URL, "perplexity/sonar")
	for _, key := range []string{"model", "instructions", "input", "stream", "tools", ""} {
		req := userReq("hi")
		req.SamplingParams = map[string]any{key: "evil"}
		before := f.hits.Load()
		if _, err := c.Complete(context.Background(), req); err == nil {
			t.Errorf("SamplingParams[%q] accepted", key)
		}
		if f.hits.Load() != before {
			t.Errorf("SamplingParams[%q]: request reached the network", key)
		}
	}
}

func TestBaseURLGetsV1OnlyWhenBareHost(t *testing.T) {
	for _, tc := range []struct{ suffix, wantPath string }{
		{"", "/v1/responses"},            // bare host gets /v1
		{"/", "/v1/responses"},           // trailing slash too
		{"/v1", "/v1/responses"},         // explicit /v1 is not doubled
		{"/custom", "/custom/responses"}, // any other path is the caller's own
	} {
		f := newFake(t, "application/json", agentJSON(10, 0))
		c := agentClient(t, f.URL+tc.suffix, "perplexity/sonar")
		if _, err := c.Complete(context.Background(), userReq("x")); err != nil {
			t.Fatalf("suffix %q: %v", tc.suffix, err)
		}
		if got := f.last(t).Path; got != tc.wantPath {
			t.Errorf("BaseURL %q + %q: path = %q, want %q", "<srv>", tc.suffix, got, tc.wantPath)
		}
	}
}

func TestAgentCachedTokenSplitAndCost(t *testing.T) {
	const model = "anthropic/claude-sonnet-5-5" // in $2.00, cache read $0.10 /1M
	f := newFake(t, "application/json", agentJSON(1_000_000, 400_000))
	c := agentClient(t, f.URL, model)
	resp, err := c.Complete(context.Background(), userReq("x"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.InputTokens != 600_000 || resp.CacheReadTokens != 400_000 {
		t.Fatalf("split = %d uncached / %d cached", resp.InputTokens, resp.CacheReadTokens)
	}
	got := llm.EstimateCost(llm.CostInput{Model: model, InputTokens: resp.InputTokens, CacheReadTokens: resp.CacheReadTokens})
	want := 0.6*2.00 + 0.4*0.10
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("cost = %v, want %v", got, want)
	}
	// Pricing the cached share as plain input would overcharge.
	if plain := llm.EstimateCost(llm.CostInput{Model: model, InputTokens: 1_000_000}); plain <= got {
		t.Errorf("cache reads not cheaper: plain %v vs %v", plain, got)
	}
}

// Hostile usage: cached > total, negative counts, missing details.
func TestAgentCachedTokensHostileUsage(t *testing.T) {
	for name, usage := range map[string]string{
		"cached exceeds total": `{"input_tokens":100,"output_tokens":1,"input_tokens_details":{"cached_tokens":900}}`,
		"negative cached":      `{"input_tokens":100,"output_tokens":1,"input_tokens_details":{"cached_tokens":-5}}`,
		"negative input":       `{"input_tokens":-100,"output_tokens":1,"input_tokens_details":{"cached_tokens":5}}`,
		"no usage at all":      `null`,
	} {
		body := `{"id":"r","status":"completed","model":"perplexity/sonar","output":[{"type":"message","content":[{"type":"output_text","text":"x"}]}],"usage":` + usage + `}`
		f := newFake(t, "application/json", body)
		c := agentClient(t, f.URL, "perplexity/sonar")
		resp, err := c.Complete(context.Background(), userReq("x"))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if resp.InputTokens < 0 || resp.CacheReadTokens < 0 || resp.InputTokens+resp.CacheReadTokens > 100 {
			t.Errorf("%s: corrupt usage %d/%d", name, resp.InputTokens, resp.CacheReadTokens)
		}
		if cost := llm.EstimateCost(llm.CostInput{Model: "perplexity/sonar", InputTokens: resp.InputTokens, CacheReadTokens: resp.CacheReadTokens}); cost < 0 || math.IsNaN(cost) {
			t.Errorf("%s: cost %v", name, cost)
		}
	}
}

func TestAgentFailedStatusIsError(t *testing.T) {
	f := newFake(t, "application/json", `{"id":"r","status":"failed","error":{"code":"bad","message":"boom"}}`)
	c := agentClient(t, f.URL, "perplexity/sonar")
	if _, err := c.Complete(context.Background(), userReq("x")); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestAgentGarbageBodyIsError(t *testing.T) {
	for _, body := range []string{"", "not json", "{", `[]`} {
		f := newFake(t, "application/json", body)
		c := agentClient(t, f.URL, "perplexity/sonar")
		if _, err := c.Complete(context.Background(), userReq("x")); err == nil {
			t.Errorf("body %q: no error", body)
		}
	}
}

// Surprise: "tools" is a reserved SamplingParams key, so Perplexity's built-in
// Agent tools (web_search, ...) cannot ride the passthrough in v0.20.0. The
// refusal is loud and happens before any request.
func TestAgentToolsViaSamplingParamsIsRefused(t *testing.T) {
	f := newFake(t, "application/json", agentJSON(10, 0))
	c := agentClient(t, f.URL, "perplexity/sonar")
	req := userReq("x")
	req.SamplingParams = map[string]any{"tools": []any{map[string]any{"type": "web_search"}}}
	_, err := c.Complete(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "reserved") || f.hits.Load() != 0 {
		t.Fatalf("err=%v hits=%d", err, f.hits.Load())
	}
}

// A trailing slash on an explicit "/v1/" BaseURL used to yield
// "/v1//responses" (found here, fixed in rho-llm v0.20.0).
func TestAgentBaseURLTrailingSlashV1(t *testing.T) {
	f := newFake(t, "application/json", agentJSON(10, 0))
	c := agentClient(t, f.URL+"/v1/", "perplexity/sonar")
	if _, err := c.Complete(context.Background(), userReq("x")); err != nil {
		t.Fatal(err)
	}
	if got := f.last(t).Path; got != "/v1/responses" {
		t.Fatalf("path = %q, want /v1/responses", got)
	}
}
