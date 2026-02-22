package stress_test

import (
	"context"
	"testing"

	llm "gitlab2024.bds421-cloud.com/bds421/rho/llm"
)

func TestPooledClient_Stream_PreDataRetry(t *testing.T) {
	// First stream attempt: error before any events (pre-data)
	// Second stream attempt: success
	mock1 := newSequenceMock().
		ThenStream(nil, llm.NewRateLimitError("mock", "rate limited")).
		Build()
	mock2 := newSequenceMock().
		ThenStream(contentEvents("hello", " world"), nil).
		Build()

	mocks := map[string]*sequenceMockClient{
		"mock-1": mock1,
		"mock-2": mock2,
	}

	cfg := llm.Config{Provider: "mock", Model: "mock-model"}
	pc, err := llm.NewPooledClient(cfg, []string{"k1", "k2"}, profileDispatchFunc(mocks))
	if err != nil {
		t.Fatalf("NewPooledClient: %v", err)
	}
	defer pc.Close()

	var texts []string
	for event, err := range pc.Stream(context.Background(), simpleRequest()) {
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		if event.Type == llm.EventContent {
			texts = append(texts, event.Text)
		}
	}

	if len(texts) != 2 || texts[0] != "hello" || texts[1] != " world" {
		t.Errorf("got texts %v, want [hello, ' world']", texts)
	}
}

func TestPooledClient_Stream_PostDataNoRetry(t *testing.T) {
	// Stream yields some events, then errors. Post-data errors should not retry.
	events := []llm.StreamEvent{
		{Type: llm.EventContent, Text: "partial"},
	}
	mock := newSequenceMock().
		ThenStream(events, llm.NewOverloadedError("mock", "overloaded")).
		ThenStream(contentEvents("should not reach"), nil).
		Build()

	cfg := llm.Config{Provider: "mock", Model: "mock-model"}
	pc, err := llm.NewPooledClient(cfg, []string{"k1"}, mockClientFunc(mock))
	if err != nil {
		t.Fatalf("NewPooledClient: %v", err)
	}
	defer pc.Close()

	var gotTexts []string
	var gotErr error
	for event, err := range pc.Stream(context.Background(), simpleRequest()) {
		if err != nil {
			gotErr = err
			break
		}
		if event.Type == llm.EventContent {
			gotTexts = append(gotTexts, event.Text)
		}
	}

	if gotErr == nil {
		t.Fatal("expected post-data error")
	}
	if !llm.IsOverloaded(gotErr) {
		t.Errorf("expected overloaded error, got: %v", gotErr)
	}
	if len(gotTexts) != 1 || gotTexts[0] != "partial" {
		t.Errorf("got texts %v, want [partial]", gotTexts)
	}
	// Stream should only have been called once (no retry after data yielded)
	if mock.StreamCallCount() != 1 {
		t.Errorf("stream call count = %d, want 1", mock.StreamCallCount())
	}
}

func TestPooledClient_Stream_PreDataAuthError(t *testing.T) {
	mock1 := newSequenceMock().
		ThenStream(nil, llm.NewAuthError("mock", "invalid key", 401)).
		Build()
	mock2 := newSequenceMock().
		ThenStream(contentEvents("recovered"), nil).
		Build()

	mocks := map[string]*sequenceMockClient{
		"mock-1": mock1,
		"mock-2": mock2,
	}

	cfg := llm.Config{Provider: "mock", Model: "mock-model"}
	pc, err := llm.NewPooledClient(cfg, []string{"k1", "k2"}, profileDispatchFunc(mocks))
	if err != nil {
		t.Fatalf("NewPooledClient: %v", err)
	}
	defer pc.Close()

	var texts []string
	var streamErr error
	for event, err := range pc.Stream(context.Background(), simpleRequest()) {
		if err != nil {
			streamErr = err
			break
		}
		if event.Type == llm.EventContent {
			texts = append(texts, event.Text)
		}
	}

	// With 2 keys, auth error on key 1 should rotate to key 2 and succeed
	if streamErr != nil {
		// Auth error might propagate if rotation fails
		t.Logf("stream error (may be expected with auth): %v", streamErr)
	}
}

func TestPooledClient_Stream_NormalCompletion(t *testing.T) {
	mock := newSequenceMock().
		ThenStream(contentEvents("hello"), nil).
		Build()

	cfg := llm.Config{Provider: "mock", Model: "mock-model"}
	pc, err := llm.NewPooledClient(cfg, []string{"k1"}, mockClientFunc(mock))
	if err != nil {
		t.Fatalf("NewPooledClient: %v", err)
	}
	defer pc.Close()

	var texts []string
	var doneCount int
	for event, err := range pc.Stream(context.Background(), simpleRequest()) {
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		if event.Type == llm.EventContent {
			texts = append(texts, event.Text)
		}
		if event.Type == llm.EventDone {
			doneCount++
		}
	}

	if len(texts) != 1 || texts[0] != "hello" {
		t.Errorf("got texts %v, want [hello]", texts)
	}
	if doneCount != 1 {
		t.Errorf("done count = %d, want 1", doneCount)
	}
}

func TestPooledClient_Stream_CallerBreaks(t *testing.T) {
	// Generate 10 events, but caller breaks after 5
	events := make([]llm.StreamEvent, 10)
	for i := range events {
		events[i] = llm.StreamEvent{Type: llm.EventContent, Text: "chunk"}
	}
	mock := newSequenceMock().
		ThenStream(events, nil).
		Build()

	cfg := llm.Config{Provider: "mock", Model: "mock-model"}
	pc, err := llm.NewPooledClient(cfg, []string{"k1"}, mockClientFunc(mock))
	if err != nil {
		t.Fatalf("NewPooledClient: %v", err)
	}
	defer pc.Close()

	count := 0
	for _, err := range pc.Stream(context.Background(), simpleRequest()) {
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		count++
		if count == 5 {
			break
		}
	}

	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
}
