package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
)

// TestLiveMessageThreads runs three real turns with threads on and checks that
// turns 2 and 3 continued the thread (one message on the wire, no system, no
// tools) and that the model still remembers turn 1. Skipped under -short or
// without ANTHROPIC_API_KEY.
func TestLiveMessageThreads(t *testing.T) {
	if testing.Short() {
		t.Skip("live test skipped in -short mode")
	}
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, rec, err := newRecordedClient(key, "")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	sess := llm.NewSession(client, llm.WithMessageThreads(), llm.WithSystem(systemPrompt), llm.WithTools(noopTool))

	if _, err := sess.Send(ctx, "Remember the number 7421. Reply with just: ok."); err != nil {
		t.Fatalf("turn 1: %v", err)
	}
	r2, err := sess.Send(ctx, "Which number did I ask you to remember? Reply with the number only.")
	if err != nil {
		t.Fatalf("turn 2: %v", err)
	}
	if !strings.Contains(r2.Content, "7421") {
		t.Errorf("turn 2 lost the conversation: %q", r2.Content)
	}
	var r3 strings.Builder
	for ev, err := range sess.Stream(ctx, "Add 1 to that number. Reply with the number only.") {
		if err != nil {
			t.Fatalf("turn 3: %v", err)
		}
		if ev.Type == llm.EventContent {
			r3.WriteString(ev.Text)
		}
	}
	if !strings.Contains(r3.String(), "7422") {
		t.Errorf("turn 3 lost the conversation: %q", r3.String())
	}

	calls := rec.calls()
	if len(calls) != 3 {
		t.Fatalf("%d HTTP requests, want 3 (no replay expected): %+v", len(calls), calls)
	}
	if calls[0].Thread != "create" || !calls[0].System || !calls[0].Tools {
		t.Errorf("turn 1 should be a full create: %s", calls[0])
	}
	for i, c := range calls[1:] {
		if c.Thread != "continue" || c.PrevID == "" || c.Messages != 1 || c.System || c.Tools {
			t.Errorf("turn %d should be a one-message continue without system/tools: %s", i+2, c)
		}
	}
	if n := len(sess.Conversation().Messages); n != 6 {
		t.Errorf("transcript has %d messages, want 6", n)
	}
}
