package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	llm "github.com/bds421/rho-llm"
)

// pngData is a valid 1x1 PNG, base64.
const pngData = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

// fake is an httptest server that records every request and answers a valid
// response for whichever wire protocol the path belongs to.
type fake struct {
	srv   *httptest.Server
	hits  atomic.Int32
	mu    sync.Mutex
	raws  [][]byte
	paths []string
	// reply picks the answer text from the request body; default PELICAN.
	reply func(body []byte) string
}

func newFake(t *testing.T) *fake {
	t.Helper()
	f := &fake{}
	f.srv = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fake) handle(w http.ResponseWriter, r *http.Request) {
	f.hits.Add(1)
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.raws = append(f.raws, body)
	f.paths = append(f.paths, r.URL.Path)
	f.mu.Unlock()
	text := "PELICAN"
	if f.reply != nil {
		text = f.reply(body)
	}
	q, _ := json.Marshal(text)
	stream := strings.Contains(string(body), `"stream":true`)
	p := r.URL.Path
	switch {
	case strings.Contains(p, "streamGenerateContent"):
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, `data: {"candidates":[{"content":{"role":"model","parts":[{"text":`+string(q)+`}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`+"\n\n")
	case strings.Contains(p, "generateContent"):
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"candidates":[{"content":{"role":"model","parts":[{"text":`+string(q)+`}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`)
	case strings.HasSuffix(p, "/chat/completions") && stream:
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, `data: {"id":"c","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":`+string(q)+`},"finish_reason":null}]}`+"\n\n")
		io.WriteString(w, `data: {"id":"c","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`+"\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	case strings.HasSuffix(p, "/chat/completions"):
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"c","object":"chat.completion","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":`+string(q)+`},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	case strings.HasSuffix(p, "/responses"):
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"resp_1","object":"response","status":"completed","model":"gpt-6-luna","output":[{"type":"message","id":"m1","role":"assistant","status":"completed","content":[{"type":"output_text","text":`+string(q)+`}]}],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`)
	default: // Anthropic messages
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"msg_1","type":"message","role":"assistant","model":"claude-haiku-5-5","content":[{"type":"text","text":`+string(q)+`}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)
	}
}

func (f *fake) count() int { return int(f.hits.Load()) }

// bodies returns every captured request body decoded as JSON.
func (f *fake) bodies(t *testing.T) []map[string]any {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []map[string]any
	for _, raw := range f.raws {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("request body is not JSON: %v\n%s", err, raw)
		}
		out = append(out, m)
	}
	return out
}

// body returns the only captured request body.
func (f *fake) body(t *testing.T) map[string]any {
	t.Helper()
	b := f.bodies(t)
	if len(b) != 1 {
		t.Fatalf("server saw %d requests, want 1", len(b))
	}
	return b[0]
}

// rewriteTransport sends every request to the fake server whatever its URL
// says, so a client can be configured with BaseURL https://api.mistral.ai/v1
// (which host detection keys on) and still talk to httptest.
type rewriteTransport struct{ to *url.URL }

func (rt rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := r.Clone(r.Context())
	r2.URL.Scheme, r2.URL.Host = rt.to.Scheme, rt.to.Host
	return http.DefaultTransport.RoundTrip(r2)
}

// customCaps declares the capabilities of a model the registry does not know
// (a custom provider name, an OpenRouter slug). rho-llm refuses to dispatch to
// a model without reviewed capability metadata, so these need an explicit
// deployment-scoped declaration. Registered models are left to the registry.
func customCaps(model string) llm.CapabilitySet {
	if _, ok := llm.GetModelInfo(model); ok {
		return 0
	}
	return llm.Capabilities(llm.CapabilityChat, llm.CapabilityStream, llm.CapabilityVision, llm.CapabilityDocumentInput)
}

// clientFor returns a client for provider/model talking to f directly.
func clientFor(t *testing.T, f *fake, provider, model string) llm.Client {
	t.Helper()
	c, err := llm.NewClient(llm.Config{Provider: provider, Model: model, APIKey: "test-key-not-real",
		BaseURL: f.srv.URL, DisableRetries: true, ModelCapabilities: customCaps(model)})
	if err != nil {
		t.Fatalf("NewClient(%s): %v", provider, err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// clientAsHost returns a client whose configured BaseURL is baseURL (a real
// vendor host) but whose traffic lands on f.
func clientAsHost(t *testing.T, f *fake, provider, model, baseURL string) llm.Client {
	t.Helper()
	u, _ := url.Parse(f.srv.URL)
	c, err := llm.NewClient(llm.Config{Provider: provider, Model: model, APIKey: "test-key-not-real",
		BaseURL: baseURL, DisableRetries: true, ModelCapabilities: customCaps(model),
		HTTPClient: &http.Client{Transport: rewriteTransport{to: u}}})
	if err != nil {
		t.Fatalf("NewClient(%s @ %s): %v", provider, baseURL, err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// testPDF is a real one-page PDF, built once.
var testPDF = buildPDF("The secret word is " + secretWord + ".")

func userMsg(parts ...llm.ContentPart) llm.Message {
	return llm.Message{Role: llm.RoleUser, Content: append([]llm.ContentPart{{Type: llm.ContentText, Text: question}}, parts...)}
}

func reqOf(msgs ...llm.Message) llm.Request { return llm.Request{MaxTokens: 100, Messages: msgs} }

func imagePart() llm.ContentPart {
	return llm.ContentPart{Type: llm.ContentImage, Source: &llm.ImageSource{Type: "base64", MediaType: "image/png", Data: pngData}}
}

func pdfDataURL(data string) string { return "data:application/pdf;base64," + data }

// chatParts returns the content arrays of every chat-completions message.
func chatParts(t *testing.T, body map[string]any) [][]map[string]any {
	t.Helper()
	var out [][]map[string]any
	for _, m := range body["messages"].([]any) {
		arr, ok := m.(map[string]any)["content"].([]any)
		if !ok {
			continue
		}
		var parts []map[string]any
		for _, p := range arr {
			parts = append(parts, p.(map[string]any))
		}
		out = append(out, parts)
	}
	return out
}

// partsOfType collects every part of the given type from the chat messages.
func partsOfType(t *testing.T, body map[string]any, typ string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, msg := range chatParts(t, body) {
		for _, p := range msg {
			if p["type"] == typ {
				out = append(out, p)
			}
		}
	}
	return out
}

// pdfAsImage returns every image_url part that carries a PDF data URL: the
// pre-v0.20.0 bug every host rejected.
func pdfAsImage(t *testing.T, body map[string]any) []string {
	t.Helper()
	var bad []string
	for _, p := range partsOfType(t, body, "image_url") {
		u, _ := p["image_url"].(map[string]any)["url"].(string)
		if strings.HasPrefix(u, "data:application/pdf") {
			bad = append(bad, u)
		}
	}
	return bad
}

func assertNoPDFAsImage(t *testing.T, body map[string]any) {
	t.Helper()
	for _, u := range pdfAsImage(t, body) {
		t.Errorf("PDF sent as image_url: %.60s...", u)
	}
}
