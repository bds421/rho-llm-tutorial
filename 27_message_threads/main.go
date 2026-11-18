// Tutorial 27: Anthropic Message Threads
//
// Demonstrates: llm.WithMessageThreads (Session option), thread create /
//               continue requests, Request.Thread / llm.Thread for direct
//               clients, Response.ID as the thread tip, llm.ThreadErrorCode
//               and the Thread* constants, anthropic.MessageThreadsBeta,
//               a wire-level recorder (http.RoundTripper) to see what travels
//
// With message threads the Anthropic API keeps the conversation for about five
// minutes. A Session turn then sends only the NEW messages plus the ID of the
// last reply, and leaves system and tools off the wire. The Session still owns
// the full transcript and falls back to a full request whenever the server
// cannot continue.
//
// Threads cut what you SEND (request size, no re-uploaded history); they do not
// change the billed input tokens, because the server rebuilds the same prompt.
//
// Requires ANTHROPIC_API_KEY. Without it the tutorial explains itself and
// exits 0.

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
	"sync"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
	"github.com/bds421/rho-llm/provider/anthropic"
)

const model = "claude-haiku-5-5"

// wireCall is what one HTTP request to the Messages API carried.
type wireCall struct {
	Messages  int
	System    bool
	Tools     bool
	Thread    string // "", "create" or "continue"
	PrevID    string
	BetaFlags string
	Bytes     int // request body size
}

func (w wireCall) String() string {
	thread := "no thread"
	switch w.Thread {
	case "create":
		thread = "thread=create"
	case "continue":
		thread = "thread=continue(prev=" + w.PrevID + ")"
	}
	return fmt.Sprintf("%d message(s), system=%-5v tools=%-5v %5d body bytes, %s", w.Messages, w.System, w.Tools, w.Bytes, thread)
}

// wireRecorder is an http.RoundTripper that records what each request body
// carried, then forwards it unchanged. Plug it in through Config.HTTPClient.
type wireRecorder struct {
	next http.RoundTripper
	mu   sync.Mutex
	got  []wireCall
}

func (r *wireRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(raw))
		r.record(raw, req.Header.Get("anthropic-beta"))
	}
	next := r.next
	if next == nil {
		next = http.DefaultTransport
	}
	return next.RoundTrip(req)
}

func (r *wireRecorder) record(raw []byte, beta string) {
	var body struct {
		Messages []json.RawMessage `json:"messages"`
		System   json.RawMessage   `json:"system"`
		Tools    json.RawMessage   `json:"tools"`
		Thread   *struct {
			Type string `json:"type"`
			Prev string `json:"previous_message_id"`
		} `json:"thread"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return
	}
	c := wireCall{Messages: len(body.Messages), System: len(body.System) > 0, Tools: len(body.Tools) > 0, BetaFlags: beta, Bytes: len(raw)}
	if body.Thread != nil {
		c.Thread, c.PrevID = body.Thread.Type, body.Thread.Prev
	}
	r.mu.Lock()
	r.got = append(r.got, c)
	r.mu.Unlock()
}

func (r *wireRecorder) calls() []wireCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]wireCall(nil), r.got...)
}

// newRecordedClient builds an Anthropic client whose HTTP traffic is recorded.
func newRecordedClient(apiKey, baseURL string) (llm.Client, *wireRecorder, error) {
	rec := &wireRecorder{}
	client, err := llm.NewClient(llm.Config{
		Provider:  "anthropic",
		Model:     model,
		APIKey:    apiKey,
		BaseURL:   baseURL,
		MaxTokens: 128,
		Timeout:   60 * time.Second,
		// One transport attempt keeps the request log equal to the turn log.
		DisableRetries: true,
		HTTPClient:     &http.Client{Transport: rec},
	})
	return client, rec, err
}

// A long, fixed system prompt and one tool: the parts a thread keeps on the
// server instead of resending every turn.
var systemPrompt = "You are a meticulous assistant. Answer in one short sentence. " +
	strings.Repeat("Prefer precise numbers over vague words. Never invent facts. ", 40)

var noopTool = llm.Tool{
	Name:        "noop",
	Description: "Does nothing. Never call it.",
	InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
}

var prompts = []string{
	"Remember the number 7421. Reply with just: ok.",
	"Which number did I ask you to remember? Reply with the number only.",
	"Add 1 to that number. Reply with the number only.",
	"Subtract 21 from your last answer. Reply with the number only.",
}

// turnReport is one Session turn as seen on the wire.
type turnReport struct {
	Prompt      string
	Reply       string
	Wire        wireCall
	PromptToks  int // input + cache read + cache creation
	OutputToks  int
	WireCalls   int // HTTP requests this turn made (1 unless a fallback replayed)
	ReplayedFor string
}

// runSession drives a Session through the prompts, with or without threads.
func runSession(ctx context.Context, apiKey, baseURL string, threads bool, ask []string) ([]turnReport, *llm.Session, error) {
	client, rec, err := newRecordedClient(apiKey, baseURL)
	if err != nil {
		return nil, nil, err
	}
	opts := []llm.SessionOption{
		llm.WithSystem(systemPrompt),
		llm.WithTools(noopTool),
	}
	if threads {
		opts = append(opts, llm.WithMessageThreads())
	}
	sess := llm.NewSession(client, opts...)

	var out []turnReport
	seen := 0
	for _, p := range ask {
		resp, err := sess.Send(ctx, p)
		if err != nil {
			return out, sess, fmt.Errorf("prompt %q: %w", p, err)
		}
		calls := rec.calls()
		turn := calls[seen:]
		seen = len(calls)
		out = append(out, turnReport{
			Prompt:     p,
			Reply:      strings.TrimSpace(resp.Content),
			Wire:       turn[len(turn)-1],
			PromptToks: resp.InputTokens + resp.CacheReadTokens + resp.CacheCreationTokens,
			OutputToks: resp.OutputTokens,
			WireCalls:  len(turn),
		})
	}
	return out, sess, nil
}

func printReports(label string, reports []turnReport) (tokens, bytes int) {
	fmt.Printf("--- %s ---\n", label)
	for i, r := range reports {
		fmt.Printf("turn %d: %s\n", i+1, r.Wire)
		fmt.Printf("        prompt tokens billed: %4d   reply: %q\n", r.PromptToks, r.Reply)
		tokens += r.PromptToks
		bytes += r.Wire.Bytes
	}
	fmt.Printf("total: %d prompt tokens billed, %d request body bytes sent\n\n", tokens, bytes)
	return tokens, bytes
}

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("Tutorial 27: Anthropic message threads")
		fmt.Println()
		fmt.Println("With llm.WithMessageThreads() a Session sends turn 1 as a full request that")
		fmt.Println(`creates a thread ("thread": {"type": "create"}); every later turn sends only`)
		fmt.Println(`the new messages plus "thread": {"type": "continue", "previous_message_id": ...}`)
		fmt.Println("and omits system and tools. The server keeps the conversation for about five")
		fmt.Println("minutes; after that, or on any thread error, the Session replays the full")
		fmt.Println("conversation. Anthropic only; other providers ignore the option.")
		fmt.Println()
		fmt.Println("Set ANTHROPIC_API_KEY to run the live demo. (Offline tests: go test -short ./...)")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// === Step 1: the same conversation without and with threads ===
	fmt.Println("=== Step 1: Session without threads (default) ===")
	plain, _, err := runSession(ctx, apiKey, "", false, prompts)
	if err != nil {
		fatal(err)
	}
	plainTokens, plainBytes := printReports("threads off", plain)

	fmt.Println("=== Step 2: Session with llm.WithMessageThreads() ===")
	threaded, sess, err := runSession(ctx, apiKey, "", true, prompts)
	if err != nil {
		fatal(err)
	}
	threadTokens, threadBytes := printReports("threads on", threaded)
	// What a thread saves is what travels, not what the model reads: the
	// server rebuilds the same prompt, so the billed input tokens match.
	fmt.Printf("billed prompt tokens: %d (plain) vs %d (threaded) - the model reads the same prompt\n", plainTokens, threadTokens)
	fmt.Printf("request bytes sent:   %d (plain) vs %d (threaded) - the saving is upload size and the growing history you resend\n\n", plainBytes, threadBytes)

	// The transcript is identical in shape either way: the Session keeps it.
	conv := sess.Conversation()
	data, _ := json.Marshal(conv)
	fmt.Printf("transcript: %d messages, %d bytes of JSON, thread-free: %v\n\n",
		len(conv.Messages), len(data), !strings.Contains(string(data), `"thread"`))

	// === Step 3: a direct client with Request.Thread ===
	fmt.Println("=== Step 3: Request.Thread on a plain client ===")
	client, rec, err := newRecordedClient(apiKey, "")
	if err != nil {
		fatal(err)
	}
	defer client.Close()
	first, err := client.Complete(ctx, llm.Request{
		System:   "Answer in one short word.",
		Messages: []llm.Message{llm.NewTextMessage(llm.RoleUser, "Pick a colour and remember it.")},
		Thread:   &llm.Thread{}, // empty PreviousMessageID creates a thread
	})
	if err != nil {
		fatal(err)
	}
	fmt.Printf("create:   reply %q, tip ID %s\n", strings.TrimSpace(first.Content), first.ID)
	second, err := client.Complete(ctx, llm.Request{
		// No System, no Tools: a continuation keeps the create request's.
		Messages: []llm.Message{llm.NewTextMessage(llm.RoleUser, "Which colour did you pick?")},
		Thread:   &llm.Thread{PreviousMessageID: first.ID},
	})
	if err != nil {
		fatal(err)
	}
	fmt.Printf("continue: reply %q\n", strings.TrimSpace(second.Content))
	for i, c := range rec.calls() {
		fmt.Printf("  wire %d: %s  beta=%q\n", i+1, c, c.BetaFlags)
	}
	fmt.Printf("  beta flag constant: %s\n\n", anthropic.MessageThreadsBeta)

	// === Step 4: inspecting thread errors ===
	fmt.Println("=== Step 4: llm.ThreadErrorCode ===")
	// Each response ID continues once, so continuing first.ID again fails.
	_, err = client.Complete(ctx, llm.Request{
		Messages: []llm.Message{llm.NewTextMessage(llm.RoleUser, "And again?")},
		Thread:   &llm.Thread{PreviousMessageID: first.ID},
	})
	reportThreadError("re-continue a used ID", err)
	_, err = client.Complete(ctx, llm.Request{
		Messages: []llm.Message{llm.NewTextMessage(llm.RoleUser, "hi")},
		Thread:   &llm.Thread{PreviousMessageID: "msg_doesnotexist0000000000"},
	})
	reportThreadError("unknown ID", err)
}

// reportThreadError prints the thread error code of err, if it has one.
func reportThreadError(what string, err error) {
	code := llm.ThreadErrorCode(err)
	switch {
	case err == nil:
		fmt.Printf("%-22s no error\n", what+":")
	case code == "":
		fmt.Printf("%-22s not a thread error: %v\n", what+":", err)
	default:
		fmt.Printf("%-22s code=%s (replayable on a Session: %v)\n", what+":", code, threadReplayable(code))
	}
}

// threadReplayable says whether a Session answers the code by replaying the
// full conversation (true) or by turning threads off (false).
func threadReplayable(code string) bool {
	return code != llm.ThreadUnsupported
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
