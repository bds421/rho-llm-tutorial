package stress_test

import (
	"context"
	"iter"
	"sync/atomic"

	llm "github.com/bds421/rho-llm"
)

// =============================================================================
// SEQUENCE MOCK CLIENT
// =============================================================================

// completionStep holds a predetermined Complete response/error pair.
type completionStep struct {
	resp *llm.Response
	err  error
}

// streamStep holds a predetermined Stream response: events to yield then an optional error.
type streamStep struct {
	events []llm.StreamEvent
	err    error
}

// sequenceMockClient replays predetermined responses in order. Thread-safe via atomic counter.
type sequenceMockClient struct {
	completeSteps []completionStep
	streamSteps   []streamStep
	completeIdx   atomic.Int32
	streamIdx     atomic.Int32
	providerName  string
	modelName     string
}

func (m *sequenceMockClient) Complete(_ context.Context, _ llm.Request) (*llm.Response, error) {
	idx := int(m.completeIdx.Add(1) - 1)
	if idx >= len(m.completeSteps) {
		idx = len(m.completeSteps) - 1 // repeat last step
	}
	step := m.completeSteps[idx]
	return step.resp, step.err
}

func (m *sequenceMockClient) Stream(_ context.Context, _ llm.Request) iter.Seq2[llm.StreamEvent, error] {
	idx := int(m.streamIdx.Add(1) - 1)
	if idx >= len(m.streamSteps) {
		idx = len(m.streamSteps) - 1
	}
	step := m.streamSteps[idx]
	return func(yield func(llm.StreamEvent, error) bool) {
		for _, ev := range step.events {
			if !yield(ev, nil) {
				return
			}
		}
		if step.err != nil {
			yield(llm.StreamEvent{}, step.err)
		}
	}
}

func (m *sequenceMockClient) Provider() string { return m.providerName }
func (m *sequenceMockClient) Model() string    { return m.modelName }
func (m *sequenceMockClient) Close() error     { return nil }

func (m *sequenceMockClient) CompleteCallCount() int {
	return int(m.completeIdx.Load())
}

func (m *sequenceMockClient) StreamCallCount() int {
	return int(m.streamIdx.Load())
}

// =============================================================================
// BUILDER PATTERN
// =============================================================================

type mockBuilder struct {
	mock *sequenceMockClient
}

func newSequenceMock() *mockBuilder {
	return &mockBuilder{
		mock: &sequenceMockClient{
			providerName: "mock",
			modelName:    "mock-model",
		},
	}
}

func (b *mockBuilder) ThenComplete(resp *llm.Response, err error) *mockBuilder {
	b.mock.completeSteps = append(b.mock.completeSteps, completionStep{resp: resp, err: err})
	return b
}

func (b *mockBuilder) ThenStream(events []llm.StreamEvent, err error) *mockBuilder {
	b.mock.streamSteps = append(b.mock.streamSteps, streamStep{events: events, err: err})
	return b
}

func (b *mockBuilder) WithProvider(name string) *mockBuilder {
	b.mock.providerName = name
	return b
}

func (b *mockBuilder) WithModel(name string) *mockBuilder {
	b.mock.modelName = name
	return b
}

func (b *mockBuilder) Build() *sequenceMockClient {
	return b.mock
}

// =============================================================================
// CLIENT FUNC HELPERS
// =============================================================================

// mockClientFunc returns a clientFunc that always returns the same mock client.
func mockClientFunc(c llm.Client) func(llm.AuthProfile) (llm.Client, error) {
	return func(_ llm.AuthProfile) (llm.Client, error) {
		return c, nil
	}
}

// profileDispatchFunc returns different mock clients based on profile.Name.
func profileDispatchFunc(mocks map[string]*sequenceMockClient) func(llm.AuthProfile) (llm.Client, error) {
	return func(profile llm.AuthProfile) (llm.Client, error) {
		if m, ok := mocks[profile.Name]; ok {
			return m, nil
		}
		// fallback: return first mock found
		for _, m := range mocks {
			return m, nil
		}
		return nil, nil
	}
}

// =============================================================================
// COMMON HELPERS
// =============================================================================

// okResponse creates a simple successful response.
func okResponse(content string) *llm.Response {
	return &llm.Response{
		ID:           "resp-ok",
		Model:        "mock-model",
		Content:      content,
		StopReason:   "end_turn",
		InputTokens:  10,
		OutputTokens: 5,
	}
}

// contentEvents creates a sequence of content streaming events ending with done.
func contentEvents(texts ...string) []llm.StreamEvent {
	events := make([]llm.StreamEvent, 0, len(texts)+1)
	for _, t := range texts {
		events = append(events, llm.StreamEvent{Type: llm.EventContent, Text: t})
	}
	events = append(events, llm.StreamEvent{Type: llm.EventDone, StopReason: "end_turn"})
	return events
}

// simpleRequest creates a minimal request for testing.
func simpleRequest() llm.Request {
	return llm.Request{
		Messages:  []llm.Message{llm.NewTextMessage(llm.RoleUser, "hello")},
		MaxTokens: 100,
	}
}
