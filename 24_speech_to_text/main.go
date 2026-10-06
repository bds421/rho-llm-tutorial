// Tutorial 24: compare speech-to-text providers side by side in the browser.
//
// One recording is sent in parallel to every configured transcription model
// through rho-llm's provider-neutral ModalityClient.TranscribeAudio. The page
// shows each transcript, its latency and an approximate cost, and — given a
// reference text — the word error rate. The server binds to localhost only.
package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	llm "github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

//go:embed index.html
var static embed.FS

const maxAudioBytes = 14 << 20 // rho-llm's Gemini inline ceiling

// candidate is one provider/model column on the page.
type candidate struct {
	ID       string `json:"id"`
	Vendor   string `json:"vendor"`
	Label    string `json:"label"`
	Provider string `json:"-"`
	Model    string `json:"model"`
	KeyEnv   string `json:"-"`
	// FullLanguage sends the BCP 47 tag ("de-AT"); otherwise only the
	// primary subtag ("de"), because OpenAI and xAI take ISO-639-1 codes.
	FullLanguage bool `json:"-"`
	// USDPerMinute is an approximate list price for the cost column; see README.
	USDPerMinute float64 `json:"usd_per_minute"`
	// Unsupported explains why a vendor has no column to run.
	Unsupported string `json:"unsupported,omitempty"`
}

var candidates = []candidate{
	{ID: "gemini-transcribe", Vendor: "Google", Label: "Gemini 3.5 Transcribe", Provider: "gemini", Model: "gemini-3.5-transcribe", KeyEnv: "GEMINI_API_KEY", FullLanguage: true, USDPerMinute: 0.005},
	{ID: "gemini-flash-lite", Vendor: "Google", Label: "Gemini 3.5 Flash Lite (multimodal prompt)", Provider: "gemini", Model: "gemini-3.5-flash-lite", KeyEnv: "GEMINI_API_KEY", FullLanguage: true, USDPerMinute: 0.0006},
	{ID: "openai-transcribe", Vendor: "OpenAI", Label: "GPT Transcribe", Provider: "openai", Model: "gpt-transcribe", KeyEnv: "OPENAI_API_KEY", USDPerMinute: 0.0045},
	{ID: "openai-mini", Vendor: "OpenAI", Label: "GPT-4o mini Transcribe", Provider: "openai", Model: "gpt-4o-mini-transcribe", KeyEnv: "OPENAI_API_KEY", USDPerMinute: 0.003},
	{ID: "xai-grok", Vendor: "xAI", Label: "Grok Voice Transcribe 2.0", Provider: "xai", Model: "grok-voice-transcribe-2.0", KeyEnv: "XAI_API_KEY", USDPerMinute: 0.0017},
	{ID: "anthropic", Vendor: "Anthropic", Label: "Claude", Unsupported: "The Claude API has no audio input and no speech-to-text endpoint."},
}

// result is one column's outcome for a recording.
type result struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	Error      string `json:"error,omitempty"`
	LatencyMS  int64  `json:"latency_ms"`
	Configured bool   `json:"configured"`
}

// transcriber is the slice of llm.ModalityClient this tutorial uses; tests
// replace it to exercise the HTTP layer without network calls.
type transcriber interface {
	TranscribeAudio(context.Context, llm.TranscriptionRequest) (string, error)
}

type server struct {
	clients map[string]transcriber
	timeout time.Duration
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8090", "listen address (keep it on localhost)")
	envFile := flag.String("env", "../.env", "dotenv file with provider API keys")
	flag.Parse()
	if err := loadDotEnv(*envFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("read %s: %v", *envFile, err)
	}
	if err := requireLoopback(*addr); err != nil {
		log.Fatal(err)
	}
	srv := &server{clients: buildClients(), timeout: 60 * time.Second}
	defer srv.close()
	for _, c := range candidates {
		state := "ready"
		if c.Unsupported != "" {
			state = "no STT API"
		} else if srv.clients[c.ID] == nil {
			state = "missing " + c.KeyEnv
		}
		log.Printf("%-10s %-32s %s", c.Vendor, c.Model, state)
	}
	log.Printf("open http://%s", *addr)
	log.Fatal((&http.Server{Addr: *addr, Handler: srv.routes(), ReadHeaderTimeout: 10 * time.Second}).ListenAndServe())
}

// requireLoopback refuses any listen address other than localhost: the page
// spends API credit and has no authentication.
func requireLoopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil || (host != "127.0.0.1" && host != "localhost" && host != "::1") {
		return fmt.Errorf("refusing to listen on %q: this page spends API credit, bind to localhost", addr)
	}
	return nil
}

func buildClients() map[string]transcriber {
	clients := map[string]transcriber{}
	for _, c := range candidates {
		key := strings.TrimSpace(os.Getenv(c.KeyEnv))
		if c.Unsupported != "" || key == "" {
			continue
		}
		client, err := llm.NewModalityClient(llm.Config{Provider: c.Provider, Model: c.Model, APIKey: key, Timeout: 60 * time.Second})
		if err != nil {
			log.Printf("%s disabled: %v", c.ID, err)
			continue
		}
		clients[c.ID] = client
	}
	return clients
}

func (s *server) close() {
	for _, c := range s.clients {
		if closer, ok := c.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /{$}", http.FileServerFS(static))
	mux.HandleFunc("GET /api/candidates", s.handleCandidates)
	mux.HandleFunc("POST /api/transcribe", s.handleTranscribe)
	return logRequests(mux)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// logRequests logs every request with its status, so a failed upload is
// visible in the log and not only in the browser.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s -> %d (%dms, %d bytes in, %s)", r.Method, r.URL.Path, rec.status,
			time.Since(start).Milliseconds(), r.ContentLength, r.Header.Get("User-Agent"))
	})
}

func (s *server) handleCandidates(w http.ResponseWriter, _ *http.Request) {
	type view struct {
		candidate
		Configured bool `json:"configured"`
	}
	out := make([]view, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, view{candidate: c, Configured: s.clients[c.ID] != nil})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) handleTranscribe(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAudioBytes+1<<20)
	if err := r.ParseMultipartForm(maxAudioBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected multipart form with an audio file (max 14 MiB)"})
		return
	}
	file, _, err := r.FormFile("audio")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing audio file"})
		return
	}
	defer file.Close()
	audio, err := io.ReadAll(io.LimitReader(file, maxAudioBytes+1))
	if err != nil || len(audio) == 0 || len(audio) > maxAudioBytes {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "audio must be 1 byte to 14 MiB"})
		return
	}
	// Trust the bytes, not the browser's Content-Type.
	mediaType := llm.AudioMediaTypeFromSignature(audio)
	if mediaType == "" {
		head := audio
		if len(head) > 12 {
			head = head[:12]
		}
		log.Printf("rejected upload: unrecognised container, %d bytes, header % x", len(audio), head)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unrecognised audio container"})
		return
	}
	language := r.FormValue("language")
	if language != "" && !languageTag.MatchString(language) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `language must be empty or a tag like "de" or "de-AT"`})
		return
	}
	vocabulary := parseVocabulary(r.FormValue("vocabulary"))
	results := s.transcribeAll(r.Context(), audio, mediaType, language, vocabulary)
	logRun(mediaType, len(audio), language, vocabulary, results)
	writeJSON(w, http.StatusOK, map[string]any{"media_type": mediaType, "results": results})
}

// transcribeAll fans one recording out to every configured column in parallel.
func (s *server) transcribeAll(ctx context.Context, audio []byte, mediaType, language string, vocabulary []string) []result {
	results := make([]result, len(candidates))
	var wg sync.WaitGroup
	for i, c := range candidates {
		results[i] = result{ID: c.ID}
		client := s.clients[c.ID]
		if client == nil {
			continue
		}
		results[i].Configured = true
		wg.Add(1)
		go func(i int, c candidate, client transcriber) {
			defer wg.Done()
			callCtx, cancel := context.WithTimeout(ctx, s.timeout)
			defer cancel()
			lang := language
			if !c.FullLanguage {
				lang, _, _ = strings.Cut(language, "-")
			}
			start := time.Now()
			text, err := client.TranscribeAudio(callCtx, llm.TranscriptionRequest{
				Model: c.Model, Audio: audio, MediaType: mediaType, Language: lang, Vocabulary: vocabulary,
			})
			results[i].LatencyMS = time.Since(start).Milliseconds()
			results[i].Text = text
			if err != nil {
				results[i].Error = err.Error()
			}
		}(i, c, client)
	}
	wg.Wait()
	return results
}

// logRun writes one block per recording so a comparison can be followed live
// (tail -f the log). Transcripts are the user's own speech on a local tool.
func logRun(mediaType string, size int, language string, vocabulary []string, results []result) {
	log.Printf("run: %s %d bytes language=%q vocabulary=%d terms", mediaType, size, language, len(vocabulary))
	for _, r := range results {
		switch {
		case !r.Configured:
			continue
		case r.Error != "":
			log.Printf("  %-18s %6dms ERROR %s", r.ID, r.LatencyMS, r.Error)
		default:
			log.Printf("  %-18s %6dms %q", r.ID, r.LatencyMS, r.Text)
		}
	}
}

// languageTag is the one shape every provider can take: "de" or "de-AT".
// Checking it once here saves a paid call per column on a doomed request.
var languageTag = regexp.MustCompile(`^[a-z]{2,3}(-[A-Z]{2})?$`)

// parseVocabulary splits one term per line (LF, CRLF or CR) or comma,
// dropping blanks.
func parseVocabulary(raw string) []string {
	var terms []string
	for _, line := range strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' }) {
		if term := strings.TrimSpace(line); term != "" {
			terms = append(terms, term)
		}
	}
	return terms
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// loadDotEnv sets KEY=VALUE pairs that are not already in the environment.
func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimPrefix(line, "export "), "=")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.Trim(strings.TrimSpace(value), `"'`)
		if _, set := os.LookupEnv(key); !set {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set %s: %w", key, err)
			}
		}
	}
	return scanner.Err()
}
