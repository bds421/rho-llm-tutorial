// Tutorial 26: PDF input across providers.
//
// Demonstrates: ContentDocument, DocumentSource, ContentPart.Title,
//               ValidateDocumentSource, RequireCapabilities with
//               CapabilityDocumentInput, and how one neutral document part is
//               translated to each provider's own wire format.
//
// The PDF is generated in code, so no testdata is needed. Providers without an
// API key in the environment are skipped. xAI is included on purpose: its Chat
// Completions API takes no files, so rho-llm refuses the PDF before any request
// is sent (live 2026-11-18). Before v0.20.0 every OpenAI-compatible host got
// the PDF as an image_url data URI and rejected it (xAI 400, Mistral 422,
// OpenAI 400).

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	llm "github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider"
)

const (
	secretWord = "PELICAN"
	question   = "What is the secret word in the document? Answer with the word only."
)

// Wire kinds: how a PDF part is encoded for each target.
const (
	wireAnthropic    = "anthropic"
	wireGemini       = "gemini"
	wireResponses    = "responses"
	wireFile         = "file"
	wireDocumentURL  = "document_url"
	wireRefused      = "refused"
	xaiWorkaroundMsg = "workaround: extract the text and send it as a text part, or send page images (xAI Chat Completions takes images, not files)"
)

// wireDescription is what the demo prints for each wire kind.
var wireDescription = map[string]string{
	wireAnthropic:   `native {"type":"document","source":{"type":"base64",...}} block`,
	wireGemini:      `{"inlineData":{"mimeType":"application/pdf","data":...}} part`,
	wireResponses:   `{"type":"input_file","filename":...,"file_data":<data URL>} item (Responses API)`,
	wireFile:        `{"type":"file","file":{"filename":...,"file_data":<data URL>}} part (Chat Completions)`,
	wireDocumentURL: `{"type":"document_url","document_url":<data URL>} part (Mistral)`,
	wireRefused:     "none: the PDF is refused before any request is sent",
}

// target is one provider/model pair of the demo.
type target struct {
	label    string
	provider string
	model    string
	keyEnv   string
	wire     string
}

var targets = []target{
	{"Anthropic", "anthropic", "claude-haiku-5-5", "ANTHROPIC_API_KEY", wireAnthropic},
	{"Gemini", "gemini", "gemini-3.8-flash", "GEMINI_API_KEY", wireGemini},
	{"OpenAI (Responses)", "openai", "gpt-6-luna", "OPENAI_API_KEY", wireResponses},
	{"OpenAI (Chat Completions)", "openai", "gpt-4.1", "OPENAI_API_KEY", wireFile},
	{"Mistral", "mistral", "mistral-medium-latest", "MISTRAL_API_KEY", wireDocumentURL},
	{"xAI", "xai", "grok-4.7", "XAI_API_KEY", wireRefused},
}

// buildPDF returns a minimal valid one-page PDF whose only text is line,
// base64-encoded as DocumentSource.Data expects.
func buildPDF(line string) string {
	content := fmt.Sprintf("BT /F1 24 Tf 72 700 Td (%s) Tj ET", line)
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return base64.StdEncoding.EncodeToString([]byte(b.String()))
}

// pdfPart wraps base64 PDF data in a neutral document part. title becomes the
// filename on hosts that take one.
func pdfPart(title, data string) llm.ContentPart {
	return llm.ContentPart{
		Type:  llm.ContentDocument,
		Title: title,
		Document: &llm.DocumentSource{
			Type:      llm.DocumentSourceBase64,
			MediaType: "application/pdf",
			Data:      data,
		},
	}
}

// docRequest asks question about the document.
func docRequest(doc llm.ContentPart) llm.Request {
	return llm.Request{
		MaxTokens: 800,
		Messages: []llm.Message{{
			Role:    llm.RoleUser,
			Content: []llm.ContentPart{{Type: llm.ContentText, Text: question}, doc},
		}},
	}
}

// newClient builds a single-attempt client for a target.
func newClient(t target, apiKey string) (llm.Client, error) {
	return llm.NewClient(llm.Config{
		Provider:       t.provider,
		Model:          t.model,
		APIKey:         apiKey,
		DisableRetries: true,
	})
}

// ask sends the document to one target and returns the model's answer.
func ask(ctx context.Context, t target, apiKey string, doc llm.ContentPart) (string, error) {
	client, err := newClient(t, apiKey)
	if err != nil {
		return "", err
	}
	defer client.Close()
	resp, err := client.Complete(ctx, docRequest(doc))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

func main() {
	if err := run(context.Background(), os.Stdout, os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, out io.Writer, getenv func(string) string) error {
	// =========================================================================
	// Step 1: Build and validate the PDF
	// =========================================================================
	fmt.Fprintln(out, "=== Step 1: Build and validate a PDF ===")
	doc := pdfPart("secret.pdf", buildPDF("The secret word is "+secretWord+"."))
	if err := llm.ValidateDocumentSource(doc); err != nil {
		return fmt.Errorf("generated PDF is invalid: %w", err)
	}
	fmt.Fprintf(out, "PDF: %d base64 bytes, media type %s, title %q: valid\n\n",
		len(doc.Document.Data), doc.Document.MediaType, doc.Title)

	// =========================================================================
	// Step 2: Validation catches bad documents before any request
	// =========================================================================
	fmt.Fprintln(out, "=== Step 2: Hostile documents are rejected locally ===")
	for _, bad := range []struct {
		name string
		part llm.ContentPart
	}{
		{"nil document", llm.ContentPart{Type: llm.ContentDocument}},
		{"empty data", pdfPart("", "")},
		{"not base64", pdfPart("", "%%%not-base64%%%")},
		{"base64 but not a PDF", pdfPart("", base64.StdEncoding.EncodeToString([]byte("hello")))},
		{"wrong media type", llm.ContentPart{Type: llm.ContentDocument, Document: &llm.DocumentSource{
			Type: llm.DocumentSourceBase64, MediaType: "text/plain", Data: doc.Document.Data}}},
	} {
		fmt.Fprintf(out, "  %-22s -> %v\n", bad.name, llm.ValidateDocumentSource(bad.part))
	}
	fmt.Fprintln(out)

	// =========================================================================
	// Step 3: Send the same PDF to every provider that has a key
	// =========================================================================
	fmt.Fprintln(out, "=== Step 3: One neutral document part, one wire format per host ===")
	for _, t := range targets {
		key := getenv(t.keyEnv)
		if key == "" {
			fmt.Fprintf(out, "[%s] skipped: %s not set\n", t.label, t.keyEnv)
			continue
		}
		fmt.Fprintf(out, "[%s] %s/%s\n  wire: %s\n", t.label, t.provider, t.model, wireDescription[t.wire])
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		answer, err := ask(cctx, t, key, doc)
		cancel()
		switch {
		case t.wire == wireRefused && err != nil:
			fmt.Fprintf(out, "  refused (no request sent): %v\n  %s\n", err, xaiWorkaroundMsg)
		case t.wire == wireRefused:
			fmt.Fprintf(out, "  UNEXPECTED: xAI accepted a PDF and answered %q\n", answer)
		case err != nil:
			fmt.Fprintf(out, "  error: %v\n", err)
		default:
			fmt.Fprintf(out, "  answer: %q (contains %s: %v)\n", answer, secretWord,
				strings.Contains(strings.ToUpper(answer), secretWord))
		}
	}
	fmt.Fprintln(out)

	// =========================================================================
	// Step 4: Ask the capability table up front
	// =========================================================================
	fmt.Fprintln(out, "=== Step 4: RequireCapabilities(CapabilityDocumentInput) ===")
	for _, t := range targets {
		err := llm.RequireCapabilities(llm.Config{Provider: t.provider, Model: t.model}, llm.CapabilityDocumentInput)
		fmt.Fprintf(out, "  %-26s document input: %v\n", t.label, err == nil)
	}
	return nil
}
