package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	llm "github.com/bds421/rho-llm"
)

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	return c
}

// --- OpenAI-compatible wire shapes ------------------------------------------

func TestOpenAICompatDocumentPartPerHost(t *testing.T) {
	uri := pdfDataURL(testPDF)
	for _, tc := range []struct {
		name, provider, model, baseURL, wantType string
	}{
		{"openai chat", "openai", "gpt-4.1", "", "file"},
		{"openrouter", "openrouter", "openai/gpt-4.1", "", "file"},
		{"unknown host", "acme", "m", "", "file"},
		{"mistral by name", "mistral", "mistral-medium-latest", "", "document_url"},
		// Host detection by BaseURL: the traffic is redirected to httptest by a
		// transport, but the configured BaseURL names the real vendor host.
		{"custom name on api.mistral.ai", "my-mistral", "m", "https://api.mistral.ai/v1", "document_url"},
		{"custom name on api.openai.com", "my-openai", "m", "https://api.openai.com/v1", "file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t)
			var c llm.Client
			if tc.baseURL != "" {
				c = clientAsHost(t, f, tc.provider, tc.model, tc.baseURL)
			} else {
				c = clientFor(t, f, tc.provider, tc.model)
			}
			resp, err := c.Complete(ctx(t), reqOf(userMsg(
				pdfPart("invoice.pdf", testPDF), imagePart(), pdfPart("", testPDF))))
			if err != nil {
				t.Fatal(err)
			}
			if resp.Content != "PELICAN" {
				t.Errorf("content %q", resp.Content)
			}
			body := f.body(t)
			assertNoPDFAsImage(t, body)

			// Images next to PDFs stay image_url.
			imgs := partsOfType(t, body, "image_url")
			if len(imgs) != 1 || !strings.HasPrefix(imgs[0]["image_url"].(map[string]any)["url"].(string), "data:image/png;base64,") {
				t.Errorf("image parts = %v, want exactly the one PNG", imgs)
			}
			docs := partsOfType(t, body, tc.wantType)
			if len(docs) != 2 {
				t.Fatalf("%d %q parts, want 2; message: %v", len(docs), tc.wantType, chatParts(t, body))
			}
			if tc.wantType == "document_url" {
				for _, d := range docs {
					if d["document_url"] != uri {
						t.Errorf("document_url = %v", d["document_url"])
					}
					// Mistral's part has no filename slot: Title is not sent.
					if _, has := d["file"]; has {
						t.Errorf("document_url part carries a file object: %v", d)
					}
				}
				return
			}
			wantNames := []string{"invoice.pdf", "document-2.pdf"} // default is numbered by PDF position
			for i, d := range docs {
				file := d["file"].(map[string]any)
				if file["file_data"] != uri || file["filename"] != wantNames[i] {
					t.Errorf("doc %d file = %v, want filename %s", i, file, wantNames[i])
				}
			}
			if len(partsOfType(t, body, "document_url")) != 0 {
				t.Error("file host also got document_url parts")
			}
		})
	}
}

func TestOpenAICompatStreamDocumentShape(t *testing.T) {
	f := newFake(t)
	c := clientFor(t, f, "openai", "gpt-4.1")
	var text strings.Builder
	for ev, err := range c.Stream(ctx(t), reqOf(userMsg(pdfPart("s.pdf", testPDF)))) {
		if err != nil {
			t.Fatal(err)
		}
		text.WriteString(ev.Text)
	}
	if text.String() != "PELICAN" {
		t.Errorf("streamed %q", text.String())
	}
	body := f.body(t)
	assertNoPDFAsImage(t, body)
	if d := partsOfType(t, body, "file"); len(d) != 1 || d[0]["file"].(map[string]any)["filename"] != "s.pdf" {
		t.Errorf("stream file parts = %v", d)
	}
}

// --- xAI refusal -------------------------------------------------------------

func TestXAIRefusesPDFWithoutRequest(t *testing.T) {
	for _, tc := range []struct{ name, provider, baseURL string }{
		{"xai", "xai", ""},
		{"grok alias", "grok", ""},
		{"custom name on api.x.ai", "my-grok", "https://api.x.ai/v1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t)
			var c llm.Client
			if tc.baseURL != "" {
				c = clientAsHost(t, f, tc.provider, "grok-custom", tc.baseURL)
			} else {
				c = clientFor(t, f, tc.provider, "grok-4.7")
			}
			req := reqOf(userMsg(pdfPart("a.pdf", testPDF)))
			if resp, err := c.Complete(ctx(t), req); err == nil {
				t.Errorf("Complete accepted a PDF: %q", resp.Content)
			} else {
				// Two layers can refuse. For registered xAI models the capability
				// table says "does not support document_input"; for a custom
				// name on api.x.ai (capabilities declared by the caller) the
				// adapter itself says "PDF documents are not supported".
				t.Logf("refusal: %v", err)
				if !strings.Contains(err.Error(), "document_input") && !strings.Contains(err.Error(), "PDF") {
					t.Errorf("refusal does not explain itself: %v", err)
				}
			}
			streamed := false
			for _, err := range c.Stream(ctx(t), req) {
				streamed = true
				if err == nil {
					t.Error("Stream yielded a non-error event for a PDF")
				}
			}
			if !streamed {
				t.Error("Stream yielded nothing: the refusal was swallowed")
			}
			if n := f.count(); n != 0 {
				t.Errorf("%d requests reached the server, want 0", n)
			}
			// The same host still serves a text-only request.
			if _, err := c.Complete(ctx(t), reqOf(userMsg())); err != nil {
				t.Errorf("text-only request on xAI failed: %v", err)
			}
			if f.count() != 1 {
				t.Errorf("text-only request count = %d, want 1", f.count())
			}
		})
	}
}

func TestXAIRefusalIsPerPDFNotPerHistory(t *testing.T) {
	// A PDF buried in an old turn poisons the whole conversation: xAI refuses it
	// before the request, even though the latest turn is text.
	f := newFake(t)
	c := clientFor(t, f, "xai", "grok-4.7")
	_, err := c.Complete(ctx(t), reqOf(
		userMsg(pdfPart("old.pdf", testPDF)),
		llm.NewTextMessage(llm.RoleAssistant, "ok"),
		llm.NewTextMessage(llm.RoleUser, "and now?")))
	if err == nil || f.count() != 0 {
		t.Errorf("err=%v hits=%d, want refusal with 0 hits", err, f.count())
	}
}

func TestCapabilityTableDocumentInput(t *testing.T) {
	want := map[string]bool{"anthropic": true, "gemini": true, "openai": true, "mistral": true, "xai": false}
	for _, tr := range targets {
		err := llm.RequireCapabilities(llm.Config{Provider: tr.provider, Model: tr.model}, llm.CapabilityDocumentInput)
		if got := err == nil; got != want[tr.provider] {
			t.Errorf("%s/%s document input = %v (err %v), want %v", tr.provider, tr.model, got, err, want[tr.provider])
		}
		// RequestCapabilities must derive the requirement from the request itself.
		req := reqOf(userMsg(pdfPart("a.pdf", testPDF)))
		found := false
		for _, c := range llm.RequestCapabilities(req, false) {
			if c == llm.CapabilityDocumentInput {
				found = true
			}
		}
		if !found {
			t.Error("RequestCapabilities does not list document input for a PDF request")
		}
	}
}

// --- Native adapters ----------------------------------------------------------

func TestAnthropicDocumentBlock(t *testing.T) {
	f := newFake(t)
	c := clientFor(t, f, "anthropic", "claude-haiku-5-5")
	resp, err := c.Complete(ctx(t), reqOf(userMsg(pdfPart("secret.pdf", testPDF), llm.NewTextDocument("notes", "plain body"))))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "PELICAN" {
		t.Errorf("content %q", resp.Content)
	}
	blocks := f.body(t)["messages"].([]any)[0].(map[string]any)["content"].([]any)
	var docs []map[string]any
	for _, b := range blocks {
		if m := b.(map[string]any); m["type"] == "document" {
			docs = append(docs, m)
		}
	}
	if len(docs) != 2 {
		t.Fatalf("%d document blocks, want 2: %v", len(docs), blocks)
	}
	pdf := docs[0]["source"].(map[string]any)
	if pdf["type"] != "base64" || pdf["media_type"] != "application/pdf" || pdf["data"] != testPDF || docs[0]["title"] != "secret.pdf" {
		t.Errorf("pdf block = %v", docs[0])
	}
	// Text documents are Anthropic-only and travel as a text source.
	txt := docs[1]["source"].(map[string]any)
	if txt["type"] != "text" || txt["media_type"] != "text/plain" || txt["data"] != "plain body" || docs[1]["title"] != "notes" {
		t.Errorf("text block = %v", docs[1])
	}
}

func TestGeminiInlineData(t *testing.T) {
	f := newFake(t)
	c := clientFor(t, f, "gemini", "gemini-3.8-flash")
	if _, err := c.Complete(ctx(t), reqOf(userMsg(pdfPart("ignored-title.pdf", testPDF)))); err != nil {
		t.Fatal(err)
	}
	parts := f.body(t)["contents"].([]any)[0].(map[string]any)["parts"].([]any)
	var inline []map[string]any
	for _, p := range parts {
		if in, ok := p.(map[string]any)["inlineData"].(map[string]any); ok {
			inline = append(inline, in)
		}
	}
	if len(inline) != 1 || inline[0]["mimeType"] != "application/pdf" || inline[0]["data"] != testPDF {
		t.Fatalf("inlineData = %v", inline)
	}
	// Gemini has no filename slot: Title does not appear anywhere in the body.
	if bytes.Contains(f.raws[0], []byte("ignored-title")) {
		t.Error("Gemini request carries the Title; it has no field for it")
	}
}

func TestResponsesInputFile(t *testing.T) {
	f := newFake(t)
	c := clientFor(t, f, "openai", "gpt-6-luna") // gpt-6-* auto-selects openai_responses
	if _, err := c.Complete(ctx(t), reqOf(userMsg(pdfPart("report.pdf", testPDF), pdfPart("", testPDF)))); err != nil {
		t.Fatal(err)
	}
	if p := f.paths[0]; !strings.HasSuffix(p, "/responses") {
		t.Fatalf("gpt-6-luna went to %s, want .../responses", p)
	}
	var files []map[string]any
	for _, in := range f.body(t)["input"].([]any) {
		content, _ := in.(map[string]any)["content"].([]any)
		for _, p := range content {
			if m := p.(map[string]any); m["type"] == "input_file" {
				files = append(files, m)
			}
		}
	}
	if len(files) != 2 {
		t.Fatalf("%d input_file items, want 2", len(files))
	}
	if files[0]["filename"] != "report.pdf" || files[0]["file_data"] != pdfDataURL(testPDF) || files[1]["filename"] != "document-2.pdf" {
		t.Errorf("input_file items = %v", files)
	}
}

// Every target in the demo table must hit the wire its description claims; this
// ties the printed table to what is actually sent.
func TestTargetsTableMatchesWire(t *testing.T) {
	for _, tr := range targets {
		t.Run(tr.label, func(t *testing.T) {
			f := newFake(t)
			c := clientFor(t, f, tr.provider, tr.model)
			_, err := c.Complete(ctx(t), docRequest(pdfPart("t.pdf", testPDF)))
			if tr.wire == wireRefused {
				if err == nil || f.count() != 0 {
					t.Fatalf("err=%v hits=%d, want refusal with no request", err, f.count())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			raw := string(f.raws[0])
			marker := map[string]string{
				wireAnthropic:   `"type":"document"`,
				wireGemini:      `"inlineData"`,
				wireResponses:   `"type":"input_file"`,
				wireFile:        `"type":"file"`,
				wireDocumentURL: `"type":"document_url"`,
			}[tr.wire]
			if !strings.Contains(raw, marker) {
				t.Errorf("%s body lacks %s: %.300s", tr.label, marker, raw)
			}
			if strings.Contains(raw, "image_url") {
				t.Errorf("%s sent an image_url for a PDF", tr.label)
			}
		})
	}
}

// --- Hostile inputs -----------------------------------------------------------

func hostileParts() map[string]llm.ContentPart {
	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	mk := func(typ, media, data string) llm.ContentPart {
		return llm.ContentPart{Type: llm.ContentDocument, Document: &llm.DocumentSource{Type: typ, MediaType: media, Data: data}}
	}
	return map[string]llm.ContentPart{
		"nil document":               {Type: llm.ContentDocument},
		"empty data":                 mk("base64", "application/pdf", ""),
		"corrupt base64":             mk("base64", "application/pdf", "%%%not base64%%%"),
		"truncated padding":          mk("base64", "application/pdf", strings.TrimRight(testPDF, "=")),
		"newline in base64":          mk("base64", "application/pdf", testPDF[:20]+"\n"+testPDF[20:]),
		"base64 text not a PDF":      mk("base64", "application/pdf", b64("hello world")),
		"PNG bytes labelled PDF":     mk("base64", "application/pdf", pngData),
		"PDF bytes labelled text":    mk("base64", "text/plain", testPDF),
		"PDF bytes labelled png":     mk("base64", "image/png", testPDF),
		"empty media type":           mk("base64", "", testPDF),
		"unknown source type":        mk("url", "application/pdf", testPDF),
		"empty source type":          mk("", "application/pdf", testPDF),
		"signature after leading ws": mk("base64", "application/pdf", b64(" \n%PDF-1.4")),
	}
}

// Everything in hostileParts is rejected locally on every wire protocol, with
// zero requests sent. The one exception is recorded below.
func TestHostileDocumentsRejectedBeforeRequest(t *testing.T) {
	providers := []struct{ provider, model string }{
		{"anthropic", "claude-haiku-5-5"}, {"gemini", "gemini-3.8-flash"},
		{"openai", "gpt-6-luna"}, {"openai", "gpt-4.1"}, {"mistral", "mistral-medium-latest"}, {"acme", "m"},
	}
	for name, part := range hostileParts() {
		for _, p := range providers {
			t.Run(name+"/"+p.provider+"/"+p.model, func(t *testing.T) {
				f := newFake(t)
				c := clientFor(t, f, p.provider, p.model)
				_, err := c.Complete(ctx(t), reqOf(userMsg(part)))
				if err == nil {
					t.Errorf("hostile document accepted (hits=%d)", f.count())
				}
				if f.count() != 0 {
					t.Errorf("%d requests sent for a rejected document", f.count())
				}
				// Streaming must refuse as well, not send half a request.
				for _, err := range c.Stream(ctx(t), reqOf(userMsg(part))) {
					if err == nil {
						t.Error("Stream accepted the hostile document")
					}
				}
				if f.count() != 0 {
					t.Errorf("%d requests sent by Stream", f.count())
				}
			})
		}
	}
}

func TestValidateDocumentSourceTable(t *testing.T) {
	if err := llm.ValidateDocumentSource(pdfPart("x", testPDF)); err != nil {
		t.Fatalf("generated PDF rejected: %v", err)
	}
	for name, part := range hostileParts() {
		if err := llm.ValidateDocumentSource(part); err == nil {
			t.Errorf("%s: ValidateDocumentSource accepted it", name)
		}
	}
	// Surprising-but-true: the check is a signature check only. A garbage body
	// with a "%PDF-" header and canonical base64 passes; the provider, not
	// rho-llm, decides whether it parses.
	if err := llm.ValidateDocumentSource(pdfPart("", base64.StdEncoding.EncodeToString([]byte("%PDF-")))); err != nil {
		t.Errorf("bare %%PDF- signature rejected (behaviour changed): %v", err)
	}
	garbage := base64.StdEncoding.EncodeToString([]byte("%PDF-garbage garbage garbage"))
	if err := llm.ValidateDocumentSource(pdfPart("", garbage)); err != nil {
		t.Errorf("signature-only PDF rejected (behaviour changed): %v", err)
	}
}

func TestTextDocumentsAreAnthropicOnly(t *testing.T) {
	doc := llm.NewTextDocument("notes", "some text")
	for _, p := range []struct{ provider, model string }{
		{"gemini", "gemini-3.8-flash"}, {"openai", "gpt-6-luna"}, {"openai", "gpt-4.1"},
		{"mistral", "mistral-medium-latest"}, {"acme", "m"},
	} {
		f := newFake(t)
		c := clientFor(t, f, p.provider, p.model)
		if _, err := c.Complete(ctx(t), reqOf(userMsg(doc))); err == nil {
			t.Errorf("%s/%s accepted a text document", p.provider, p.model)
		}
		if f.count() != 0 {
			t.Errorf("%s/%s sent %d requests for a text document", p.provider, p.model, f.count())
		}
	}
	// ...and Anthropic accepts it, including an empty-title and unicode body.
	f := newFake(t)
	c := clientFor(t, f, "anthropic", "claude-haiku-5-5")
	if _, err := c.Complete(ctx(t), reqOf(userMsg(llm.NewTextDocument("", "Grüße 🦩 \"quoted\"")))); err != nil {
		t.Fatalf("anthropic text document: %v", err)
	}
	if !strings.Contains(string(f.raws[0]), "Grüße") {
		t.Error("unicode text document was mangled")
	}
	// An empty text document is rejected locally.
	if _, err := c.Complete(ctx(t), reqOf(userMsg(llm.NewTextDocument("t", "")))); err == nil {
		t.Error("empty text document accepted")
	}
}

func TestManyPDFsInOneMessage(t *testing.T) {
	const n = 60
	var parts []llm.ContentPart
	for i := 1; i <= n; i++ {
		parts = append(parts, pdfPart(fmt.Sprintf("doc-%d.pdf", i), testPDF))
	}
	for _, tc := range []struct{ provider, model, typ string }{
		{"openai", "gpt-4.1", "file"}, {"mistral", "mistral-medium-latest", "document_url"},
	} {
		f := newFake(t)
		c := clientFor(t, f, tc.provider, tc.model)
		if _, err := c.Complete(ctx(t), reqOf(userMsg(parts...))); err != nil {
			t.Fatalf("%s: %v", tc.provider, err)
		}
		docs := partsOfType(t, f.body(t), tc.typ)
		if len(docs) != n {
			t.Fatalf("%s: %d parts, want %d", tc.provider, len(docs), n)
		}
		if tc.typ == "file" {
			for i, d := range docs { // order must be preserved
				if got := d["file"].(map[string]any)["filename"]; got != fmt.Sprintf("doc-%d.pdf", i+1) {
					t.Fatalf("part %d filename %v: order lost", i, got)
				}
			}
		}
	}
	// Anthropic and Gemini likewise keep all of them.
	fa := newFake(t)
	if _, err := clientFor(t, fa, "anthropic", "claude-haiku-5-5").Complete(ctx(t), reqOf(userMsg(parts...))); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(fa.raws[0]), `"type":"document"`); got != n {
		t.Errorf("anthropic sent %d document blocks, want %d", got, n)
	}
	fg := newFake(t)
	if _, err := clientFor(t, fg, "gemini", "gemini-3.8-flash").Complete(ctx(t), reqOf(userMsg(parts...))); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(fg.raws[0]), `"inlineData"`); got != n {
		t.Errorf("gemini sent %d inlineData parts, want %d", got, n)
	}
}

func TestPDFsInSeveralHistoryTurns(t *testing.T) {
	history := reqOf(
		userMsg(pdfPart("one.pdf", testPDF)),
		llm.NewTextMessage(llm.RoleAssistant, "first answer"),
		userMsg(pdfPart("two.pdf", testPDF), imagePart()),
		llm.NewTextMessage(llm.RoleAssistant, "second answer"),
		userMsg(pdfPart("three.pdf", testPDF)),
	)
	f := newFake(t)
	if _, err := clientFor(t, f, "openai", "gpt-4.1").Complete(ctx(t), history); err != nil {
		t.Fatal(err)
	}
	body := f.body(t)
	assertNoPDFAsImage(t, body)
	var names []string
	for _, d := range partsOfType(t, body, "file") {
		names = append(names, d["file"].(map[string]any)["filename"].(string))
	}
	// Surprising-but-true: the default-name counter and Title are per message,
	// so each turn's first untitled PDF would be document-1.pdf; titled ones
	// keep their names, in order.
	if strings.Join(names, ",") != "one.pdf,two.pdf,three.pdf" {
		t.Errorf("filenames = %v", names)
	}
	if n := len(partsOfType(t, body, "image_url")); n != 1 {
		t.Errorf("%d image parts, want 1", n)
	}
	// Mistral: all three documents, all document_url.
	fm := newFake(t)
	if _, err := clientFor(t, fm, "mistral", "mistral-medium-latest").Complete(ctx(t), history); err != nil {
		t.Fatal(err)
	}
	if n := len(partsOfType(t, fm.body(t), "document_url")); n != 3 {
		t.Errorf("mistral: %d document_url parts, want 3", n)
	}
	// Untitled PDFs in separate turns restart the default counter per message.
	untitled := reqOf(userMsg(pdfPart("", testPDF)), llm.NewTextMessage(llm.RoleAssistant, "a"), userMsg(pdfPart("", testPDF)))
	fu := newFake(t)
	if _, err := clientFor(t, fu, "openai", "gpt-4.1").Complete(ctx(t), untitled); err != nil {
		t.Fatal(err)
	}
	var un []string
	for _, d := range partsOfType(t, fu.body(t), "file") {
		un = append(un, d["file"].(map[string]any)["filename"].(string))
	}
	if strings.Join(un, ",") != "document-1.pdf,document-1.pdf" {
		t.Errorf("untitled filenames across turns = %v (duplicates are the recorded behaviour)", un)
	}
}

func TestHostileTitles(t *testing.T) {
	titles := []string{`quote".pdf`, "new\nline.pdf", "../../etc/passwd", "日本語.pdf", strings.Repeat("x", 5000), `back\slash.pdf`, "\x00nul.pdf"}
	for _, title := range titles {
		f := newFake(t)
		c := clientFor(t, f, "openai", "gpt-4.1")
		if _, err := c.Complete(ctx(t), reqOf(userMsg(pdfPart(title, testPDF)))); err != nil {
			t.Errorf("title %.20q rejected: %v", title, err)
			continue
		}
		docs := partsOfType(t, f.body(t), "file") // body() re-parses JSON: broken escaping would fail here
		// Recorded behaviour: the Title goes out verbatim as the filename. rho-llm
		// does not sanitise it (path traversal text included); callers that take
		// titles from untrusted input must clean them.
		if len(docs) != 1 || docs[0]["file"].(map[string]any)["filename"] != title {
			t.Errorf("title %.20q did not round-trip", title)
		}
	}
}

func TestLargePDFPassesThrough(t *testing.T) {
	big := base64.StdEncoding.EncodeToString(append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte{'a'}, 3<<20)...))
	f := newFake(t)
	if _, err := clientFor(t, f, "openai", "gpt-4.1").Complete(ctx(t), reqOf(userMsg(pdfPart("big.pdf", big)))); err != nil {
		t.Fatalf("4 MB PDF rejected client-side: %v", err)
	}
	d := partsOfType(t, f.body(t), "file")
	if len(d) != 1 || d[0]["file"].(map[string]any)["file_data"] != pdfDataURL(big) {
		t.Error("large PDF altered in transit")
	}
}

func TestDocumentOutsideUserTurn(t *testing.T) {
	// Recorded behaviour (surprising: it differs per adapter). A PDF in an
	// ASSISTANT message is: rejected by Anthropic (error, no request), silently
	// dropped by openai_compat and Responses (request sent without the bytes, no
	// error, even when the document is corrupt: it is not validated), and forwarded
	// as inlineData by Gemini (which does validate). It is never sent as image_url.
	corrupt := llm.ContentPart{Type: llm.ContentDocument, Document: &llm.DocumentSource{Type: "base64", MediaType: "application/pdf", Data: "%%%"}}
	for _, tc := range []struct {
		provider, model string
		wantErr         bool
		wantBytes       bool
	}{
		{"openai", "gpt-4.1", false, false},
		{"mistral", "mistral-medium-latest", false, false},
		{"openai", "gpt-6-luna", false, false},
		{"gemini", "gemini-3.8-flash", false, true},
		{"anthropic", "claude-haiku-5-5", true, false},
	} {
		for name, doc := range map[string]llm.ContentPart{"valid": pdfPart("a.pdf", testPDF), "corrupt": corrupt} {
			f := newFake(t)
			c := clientFor(t, f, tc.provider, tc.model)
			asst := llm.Message{Role: llm.RoleAssistant, Content: []llm.ContentPart{doc}}
			_, err := c.Complete(ctx(t), reqOf(userMsg(), asst, userMsg()))
			t.Logf("%s/%s %s PDF in assistant turn: err=%v hits=%d", tc.provider, tc.model, name, err, f.count())
			if tc.wantErr != (err != nil) && name == "valid" {
				t.Errorf("%s/%s valid: err=%v, want error=%v", tc.provider, tc.model, err, tc.wantErr)
			}
			for _, raw := range f.raws {
				if has := bytes.Contains(raw, []byte(testPDF)); name == "valid" && has != tc.wantBytes {
					t.Errorf("%s/%s: PDF bytes present=%v, want %v", tc.provider, tc.model, has, tc.wantBytes)
				}
				if bytes.Contains(raw, []byte(`image_url`)) && bytes.Contains(raw, []byte("application/pdf")) {
					t.Errorf("%s/%s: PDF sent as image_url", tc.provider, tc.model)
				}
			}
		}
	}
}

func TestConcurrentRequestsOnOneClient(t *testing.T) {
	// Each goroutine sends a different PDF title; the server echoes back the
	// filename it received. Any cross-talk between requests shows up as a
	// mismatched answer.
	echo := func(body []byte) string {
		var m map[string]any
		_ = json.Unmarshal(body, &m)
		msgs, _ := m["messages"].([]any)
		for _, msg := range msgs {
			arr, _ := msg.(map[string]any)["content"].([]any)
			for _, p := range arr {
				if file, ok := p.(map[string]any)["file"].(map[string]any); ok {
					return file["filename"].(string)
				}
			}
		}
		return "NOFILE"
	}
	f := newFake(t)
	f.reply = echo
	c := clientFor(t, f, "openai", "gpt-4.1")
	const workers, each = 16, 6
	var wg sync.WaitGroup
	errs := make(chan string, workers*each)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				title := fmt.Sprintf("w%d-%d.pdf", w, i)
				resp, err := c.Complete(context.Background(), reqOf(userMsg(pdfPart(title, testPDF))))
				if err != nil {
					errs <- err.Error()
				} else if resp.Content != title {
					errs <- fmt.Sprintf("sent %s, server saw %s", title, resp.Content)
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	if f.count() != workers*each {
		t.Errorf("server saw %d requests, want %d", f.count(), workers*each)
	}
}

func TestConcurrentMixOfValidAndRefused(t *testing.T) {
	// A refused request on a shared xAI client must not poison valid ones.
	f := newFake(t)
	c := clientFor(t, f, "xai", "grok-4.7")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Complete(context.Background(), reqOf(userMsg(pdfPart("a.pdf", testPDF)))); err == nil {
				t.Error("PDF accepted")
			}
			if _, err := c.Complete(context.Background(), reqOf(userMsg())); err != nil {
				t.Errorf("text request failed: %v", err)
			}
		}()
	}
	wg.Wait()
	if f.count() != 20 {
		t.Errorf("server saw %d requests, want exactly the 20 text ones", f.count())
	}
}

func TestRequestStructsAreNotMutated(t *testing.T) {
	// The request passed in must come out unchanged (the adapter must not
	// normalise Data/Title in place).
	part := pdfPart("keep.pdf", testPDF)
	req := reqOf(userMsg(part))
	before, _ := json.Marshal(req)
	f := newFake(t)
	for _, p := range []struct{ provider, model string }{{"openai", "gpt-4.1"}, {"mistral", "mistral-medium-latest"}, {"anthropic", "claude-haiku-5-5"}, {"gemini", "gemini-3.8-flash"}} {
		if _, err := clientFor(t, f, p.provider, p.model).Complete(ctx(t), req); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := json.Marshal(req)
	if !bytes.Equal(before, after) {
		t.Error("request was mutated by Complete")
	}
}

// --- The demo itself ----------------------------------------------------------

func TestRunSkipsProvidersWithoutKeys(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), &out, func(string) string { return "" }); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, tr := range targets {
		if !strings.Contains(s, "["+tr.label+"] skipped: "+tr.keyEnv+" not set") {
			t.Errorf("no skip line for %s:\n%s", tr.label, s)
		}
	}
	if !strings.Contains(s, "document input: false") || !strings.Contains(s, "document input: true") {
		t.Errorf("step 4 table missing:\n%s", s)
	}
}

func TestRunXAIRefusalNeedsNoNetwork(t *testing.T) {
	// Only an xAI key (a fake one): the demo must print the refusal and the
	// workaround without a single request leaving the process.
	var out bytes.Buffer
	getenv := func(k string) string {
		if k == "XAI_API_KEY" {
			return "fake-key-not-real"
		}
		return ""
	}
	if err := run(context.Background(), &out, getenv); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "refused (no request sent)") || !strings.Contains(s, "workaround:") || strings.Contains(s, "UNEXPECTED") {
		t.Errorf("xAI section wrong:\n%s", s)
	}
}

func TestBuildPDFIsValidAndHostileEdits(t *testing.T) {
	if err := llm.ValidateDocumentSource(pdfPart("", buildPDF("x"))); err != nil {
		t.Fatal(err)
	}
	raw, _ := base64.StdEncoding.DecodeString(testPDF)
	if !bytes.Contains(raw, []byte("PELICAN")) || !bytes.HasPrefix(raw, []byte("%PDF-")) {
		t.Error("generated PDF lacks the secret word or signature")
	}
}

// The detector itself must bite: fed the legacy (pre-v0.20.0) body shape it has
// to flag the PDF, and it must not flag a real image.
func TestPDFAsImageDetectorBites(t *testing.T) {
	legacy := `{"messages":[{"role":"user","content":[{"type":"text","text":"q"},
		{"type":"image_url","image_url":{"url":"` + pdfDataURL(testPDF) + `"}}]}]}`
	var body map[string]any
	if err := json.Unmarshal([]byte(legacy), &body); err != nil {
		t.Fatal(err)
	}
	if got := pdfAsImage(t, body); len(got) != 1 {
		t.Errorf("detector missed the legacy shape: %v", got)
	}
	ok := `{"messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,` + pngData + `"}}]}]}`
	_ = json.Unmarshal([]byte(ok), &body)
	if got := pdfAsImage(t, body); len(got) != 0 {
		t.Errorf("detector flagged a real image: %v", got)
	}
}
