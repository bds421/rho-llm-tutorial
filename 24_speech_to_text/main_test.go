package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	llm "github.com/bds421/rho-llm"
)

var wav = []byte("RIFF\x24\x00\x00\x00WAVEfmt \x10\x00\x00\x00")

type fakeTranscriber struct {
	mu   sync.Mutex
	got  []llm.TranscriptionRequest
	text string
	err  error
	wait time.Duration
}

func (f *fakeTranscriber) TranscribeAudio(ctx context.Context, req llm.TranscriptionRequest) (string, error) {
	f.mu.Lock()
	f.got = append(f.got, req)
	f.mu.Unlock()
	select {
	case <-time.After(f.wait):
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return f.text, f.err
}

func post(t *testing.T, srv *server, fields map[string]string, audio []byte) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	if audio != nil {
		fw, _ := mw.CreateFormFile("audio", "clip")
		_, _ = fw.Write(audio)
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/transcribe", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

func TestFanOutSendsLanguagePerProviderAndIsolatesFailures(t *testing.T) {
	gemini := &fakeTranscriber{text: "zwei Weckerl"}
	openai := &fakeTranscriber{err: errors.New("quota exceeded")}
	xai := &fakeTranscriber{text: "zwei Weckerl", wait: 20 * time.Millisecond}
	srv := &server{timeout: time.Second, clients: map[string]transcriber{
		"gemini-transcribe": gemini, "openai-transcribe": openai, "xai-grok": xai,
	}}
	rec, out := post(t, srv, map[string]string{"language": "de-AT", "vocabulary": "Weckerl\n, Meal Prep\n\n"}, wav)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if out["media_type"] != "audio/wav" {
		t.Fatalf("media type sniffed as %v", out["media_type"])
	}
	byID := map[string]map[string]any{}
	for _, r := range out["results"].([]any) {
		m := r.(map[string]any)
		byID[m["id"].(string)] = m
	}
	if byID["gemini-transcribe"]["text"] != "zwei Weckerl" || byID["xai-grok"]["text"] != "zwei Weckerl" {
		t.Fatalf("a failing provider must not blank the others: %v", byID)
	}
	if byID["openai-transcribe"]["error"] != "quota exceeded" {
		t.Fatalf("provider error not surfaced: %v", byID["openai-transcribe"])
	}
	if byID["anthropic"]["configured"] != false || byID["openai-mini"]["configured"] != false {
		t.Fatal("unconfigured columns must report configured=false")
	}
	if gemini.got[0].Language != "de-AT" || openai.got[0].Language != "de" || xai.got[0].Language != "de" {
		t.Fatalf("languages: gemini=%q openai=%q xai=%q", gemini.got[0].Language, openai.got[0].Language, xai.got[0].Language)
	}
	if v := gemini.got[0].Vocabulary; len(v) != 2 || v[0] != "Weckerl" || v[1] != "Meal Prep" {
		t.Fatalf("vocabulary = %q", v)
	}
}

func TestSlowProviderTimesOutWithoutBlockingTheRest(t *testing.T) {
	slow := &fakeTranscriber{text: "late", wait: time.Second}
	fast := &fakeTranscriber{text: "fast"}
	srv := &server{timeout: 50 * time.Millisecond, clients: map[string]transcriber{"gemini-transcribe": slow, "xai-grok": fast}}
	start := time.Now()
	rec, out := post(t, srv, map[string]string{"language": "de"}, wav)
	if rec.Code != http.StatusOK || time.Since(start) > 500*time.Millisecond {
		t.Fatalf("status %d after %v", rec.Code, time.Since(start))
	}
	for _, r := range out["results"].([]any) {
		m := r.(map[string]any)
		if m["id"] == "gemini-transcribe" && m["error"] == nil {
			t.Fatal("timed-out provider reported no error")
		}
	}
}

func TestRejectsBadUploadsBeforeAnyProviderCall(t *testing.T) {
	fake := &fakeTranscriber{text: "x"}
	srv := &server{timeout: time.Second, clients: map[string]transcriber{"gemini-transcribe": fake}}
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}
	for name, audio := range map[string][]byte{"no file": nil, "empty file": {}, "png": png, "text": []byte("hello")} {
		if rec, _ := post(t, srv, nil, audio); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d", name, rec.Code)
		}
	}
	oversized := append(append([]byte(nil), wav...), make([]byte, maxAudioBytes)...)
	if rec, _ := post(t, srv, nil, oversized); rec.Code != http.StatusBadRequest {
		t.Errorf("oversized: status %d", rec.Code)
	}
	if len(fake.got) != 0 {
		t.Fatalf("%d provider call(s) for invalid uploads", len(fake.got))
	}
}

func TestRequireLoopback(t *testing.T) {
	for _, ok := range []string{"127.0.0.1:8090", "localhost:1", "[::1]:8090"} {
		if err := requireLoopback(ok); err != nil {
			t.Errorf("%s rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{":8090", "0.0.0.0:8090", "192.168.1.5:8090", "example.com:80", "garbage"} {
		if requireLoopback(bad) == nil {
			t.Errorf("%s accepted", bad)
		}
	}
}

func TestParseVocabulary(t *testing.T) {
	got := parseVocabulary(" Weckerl \r\n\nMeal Prep, Kraut ,,")
	if len(got) != 3 || got[0] != "Weckerl" || got[1] != "Meal Prep" || got[2] != "Kraut" {
		t.Fatalf("got %q", got)
	}
	if parseVocabulary("  \n , ") != nil {
		t.Fatal("blank input must give no terms")
	}
}
