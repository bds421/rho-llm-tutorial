package main

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"iter"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	llm "github.com/bds421/rho-llm"
)

type fakeClient struct {
	response *llm.Response
	request  llm.Request
	calls    int
}

func (f *fakeClient) Complete(_ context.Context, r llm.Request) (*llm.Response, error) {
	f.request = r
	f.calls++
	return f.response, nil
}
func (f *fakeClient) Stream(context.Context, llm.Request) iter.Seq2[llm.StreamEvent, error] {
	panic("unused")
}
func (f *fakeClient) Provider() string { return "fake" }
func (f *fakeClient) Model() string    { return "fake" }
func (f *fakeClient) Close() error     { return nil }

func TestEstimateValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   map[string]any
		wantErr bool
	}{
		{"valid", map[string]any{"calories_kcal": 500, "fat_g": 10, "carbs_g": 70, "protein_g": 20, "assumptions": "one plate"}, false},
		{"missing", map[string]any{"calories_kcal": 500}, true},
		{"nonfinite", map[string]any{"calories_kcal": math.Inf(1), "fat_g": 10, "carbs_g": 70, "protein_g": 20, "assumptions": "one plate"}, true},
		{"blank assumptions", map[string]any{"calories_kcal": 500, "fat_g": 10, "carbs_g": 70, "protein_g": 20, "assumptions": "  "}, true},
		{"negative", map[string]any{"calories_kcal": 500, "fat_g": -1, "carbs_g": 70, "protein_g": 20, "assumptions": "one plate"}, true},
		{"unknown", map[string]any{"calories_kcal": 500, "fat_g": 10, "carbs_g": 70, "protein_g": 20, "assumptions": "one plate", "extra": 1}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeClient{response: &llm.Response{StopReason: "tool_use", ToolCalls: []llm.ToolCall{{Name: toolName, Input: tc.input}}}}
			_, err := estimate(context.Background(), f, llm.Message{})
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v", err)
			}
			if f.calls != 1 || f.request.ToolChoice.Name != toolName || f.request.ToolChoice.Mode != llm.ToolChoiceTool {
				t.Fatal("expected one forced-tool request")
			}
		})
	}
	for _, resp := range []*llm.Response{nil, {}, {StopReason: "max_tokens", ToolCalls: []llm.ToolCall{{Name: toolName}}}, {StopReason: "tool_use", ToolCalls: []llm.ToolCall{{Name: "wrong"}}}} {
		if _, err := estimate(context.Background(), &fakeClient{response: resp}, llm.Message{}); err == nil {
			t.Fatal("accepted incomplete/wrong response")
		}
	}
}

func TestErrorText(t *testing.T) {
	if got := errorText(120, 100); got != "absolute error 20.00; percent error 20.00%" {
		t.Fatal(got)
	}
	if !strings.Contains(errorText(1, 0), "N/A") {
		t.Fatal("zero reference must not divide by zero")
	}
}

func TestImageValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.png")
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	msg, err := imageMessage(path)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content[0].Source.MediaType != "image/png" {
		t.Fatal("wrong media type")
	}
	for _, data := range [][]byte{[]byte("not image"), b.Bytes()[:20], make([]byte, maxImageBytes+1)} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := imageMessage(path); err == nil {
			t.Fatal("accepted invalid/oversized image")
		}
	}
}

func TestLocalFlagsRejectBeforeIO(t *testing.T) {
	for _, args := range [][]string{{}, {"-image", "absent", "-fat", "NaN"}, {"-image", "absent", "-calories", "-1"}, {"-image", "absent", "-timeout", "3m"}} {
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
