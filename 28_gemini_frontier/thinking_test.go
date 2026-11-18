package main

import (
	"strings"
	"testing"

	"github.com/bds421/rho-llm"
)

const okBody = `{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":1}}`

// What thinkingLevel reaches the wire, per model: xhigh exists on Hydrogen only,
// "minimal" is a 400 on 3.8 Flash / 3.7 Flash / every Gemini 4 so it becomes
// "low", and levels are never sent together with a budget.
func TestThinkingLevelOnTheWire(t *testing.T) {
	cases := []struct {
		model string
		level llm.ThinkingLevel
		want  string
	}{
		{"gemini-4-hydrogen", llm.ThinkingXHigh, "xhigh"},
		{"gemini-4-hydrogen", llm.ThinkingMax, "xhigh"},
		{"gemini-4-hydrogen", llm.ThinkingHigh, "high"},
		{"gemini-4-hydrogen", llm.ThinkingMedium, "medium"},
		{"gemini-4-hydrogen", llm.ThinkingMinimal, "low"},
		{"gemini-4-carbon", llm.ThinkingXHigh, "high"},
		{"gemini-4-carbon", llm.ThinkingMax, "high"},
		{"gemini-4-carbon", llm.ThinkingMinimal, "low"},
		{"gemini-4-argon", llm.ThinkingMinimal, "low"},
		{"gemini-4-argon", llm.ThinkingXHigh, "high"},
		{"gemini-3.8-flash", llm.ThinkingXHigh, "high"},
		{"gemini-3.8-flash", llm.ThinkingMax, "high"},
		{"gemini-3.8-flash", llm.ThinkingMinimal, "low"},
		{"gemini-3.8-flash", llm.ThinkingLow, "low"},
		{"gemini-3.7-flash", llm.ThinkingMinimal, "low"},
	}
	for _, tc := range cases {
		t.Run(tc.model+"/"+string(tc.level), func(t *testing.T) {
			f := newFake(t, jsonReply(okBody))
			c := geminiClient(t, f, tc.model)
			req := userReq("hi")
			req.ThinkingLevel = tc.level
			if _, err := c.Complete(t.Context(), req); err != nil {
				t.Fatal(err)
			}
			tcfg, _ := dig(f.req(0).Body, "generationConfig", "thinkingConfig").(map[string]any)
			if tcfg == nil {
				t.Fatalf("no thinkingConfig in %v", f.req(0).Body)
			}
			if tcfg["thinkingLevel"] != tc.want {
				t.Errorf("thinkingLevel = %v, want %q", tcfg["thinkingLevel"], tc.want)
			}
			if _, has := tcfg["thinkingBudget"]; has {
				t.Errorf("a level must not travel with thinkingBudget: %v", tcfg)
			}
			if tcfg["includeThoughts"] != true {
				t.Errorf("includeThoughts = %v, want true (otherwise no thought summaries come back)", tcfg["includeThoughts"])
			}
			wantPath := "/v1beta/models/" + tc.model + ":generateContent"
			if f.req(0).Path != wantPath || !f.req(0).HasKey {
				t.Errorf("path %q (key sent: %v), want %q", f.req(0).Path, f.req(0).HasKey, wantPath)
			}
		})
	}
}

// The same rules hold for streamGenerateContent and for a level set on Config.
func TestThinkingLevelStreamAndConfigDefault(t *testing.T) {
	f := newFake(t, sseReply(chunk("ok", "STOP", "", nil)))
	c := geminiClient(t, f, "gemini-4-carbon", func(c *llm.Config) { c.ThinkingLevel = llm.ThinkingMinimal })
	r := drain(t, c, userReq("hi")) // no level on the request: Config's applies
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if got := dig(f.req(0).Body, "generationConfig", "thinkingConfig", "thinkingLevel"); got != "low" {
		t.Errorf("thinkingLevel = %v, want low", got)
	}
	if f.req(0).Path != "/v1beta/models/gemini-4-carbon:streamGenerateContent" || f.req(0).RawQuery != "alt=sse" {
		t.Errorf("stream endpoint %q?%q", f.req(0).Path, f.req(0).RawQuery)
	}
}

// No level anywhere: no thinkingConfig at all (the model's default applies).
func TestNoThinkingConfigWithoutALevel(t *testing.T) {
	f := newFake(t, jsonReply(okBody))
	c := geminiClient(t, f, "gemini-4-hydrogen")
	if _, err := c.Complete(t.Context(), userReq("hi")); err != nil {
		t.Fatal(err)
	}
	if got := dig(f.req(0).Body, "generationConfig", "thinkingConfig"); got != nil {
		t.Errorf("thinkingConfig = %v, want absent", got)
	}
}

// A custom token budget replaces the level: the two are mutually exclusive.
func TestThinkingBudgetExcludesLevel(t *testing.T) {
	f := newFake(t, jsonReply(okBody))
	c := geminiClient(t, f, "gemini-3.8-flash")
	req := userReq("hi")
	req.ThinkingLevel = llm.ThinkingHigh
	req.ThinkingBudget = 2048
	if _, err := c.Complete(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	tcfg, _ := dig(f.req(0).Body, "generationConfig", "thinkingConfig").(map[string]any)
	if _, has := tcfg["thinkingLevel"]; has || tcfg["thinkingBudget"] != float64(2048) {
		t.Errorf("thinkingConfig = %v, want only thinkingBudget 2048", tcfg)
	}
}

// Long decode: a Gemini 4 caller's total output budget is never lowered to one
// request's limit, while a 3.x model's is clamped to its 65,536 limit.
func TestMaxOutputTokensPaddingPerModel(t *testing.T) {
	const want = 262144
	for model, check := range map[string]func(got float64) bool{
		"gemini-4-carbon":   func(g float64) bool { return g >= want },
		"gemini-4-hydrogen": func(g float64) bool { return g >= want },
		"gemini-3.8-flash":  func(g float64) bool { return g == 65536 },
	} {
		t.Run(model, func(t *testing.T) {
			f := newFake(t, jsonReply(okBody))
			c := geminiClient(t, f, model)
			req := userReq("hi")
			req.MaxTokens = want
			if _, err := c.Complete(t.Context(), req); err != nil {
				t.Fatal(err)
			}
			got, _ := dig(f.req(0).Body, "generationConfig", "maxOutputTokens").(float64)
			if !check(got) {
				t.Errorf("maxOutputTokens = %v for MaxTokens %d", got, want)
			}
		})
	}
}

// A hostile unknown Gemini 4 model: the gemini-4- prefix still brings the
// Gemini 4 rules (minimal -> low, no explicit cache) before the registry knows it.
func TestUnregisteredGemini4ModelGetsGemini4Rules(t *testing.T) {
	f := newFake(t, jsonReply(okBody))
	// The factory refuses a model with no reviewed metadata, so declare it.
	c := geminiClient(t, f, "gemini-4-neon", func(c *llm.Config) {
		c.ModelCapabilities = llm.Capabilities(llm.CapabilityChat, llm.CapabilityStream, llm.CapabilityReasoning)
	})
	req := userReq("hi")
	req.ThinkingLevel = llm.ThinkingMinimal
	if _, err := c.Complete(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	if got := dig(f.req(0).Body, "generationConfig", "thinkingConfig", "thinkingLevel"); got != "low" {
		t.Errorf("thinkingLevel = %v, want low", got)
	}
	req.CachedContent = "cachedContents/abc"
	_, err := c.Complete(t.Context(), req)
	if err == nil || !strings.Contains(err.Error(), "explicit caching") || f.hits() != 1 {
		t.Errorf("err = %v, hits %d: want the caching refusal before any request", err, f.hits())
	}
}
