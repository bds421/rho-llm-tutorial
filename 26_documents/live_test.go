package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLivePDFAcrossProviders sends the generated PDF to each provider whose key is set. One cheap
// call per target. Skipped under -short and when the key is missing.
func TestLivePDFAcrossProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("live test: skipped under -short")
	}
	doc := pdfPart("secret.pdf", testPDF)
	for _, tr := range targets {
		t.Run(tr.label, func(t *testing.T) {
			key := os.Getenv(tr.keyEnv)
			if key == "" {
				t.Skipf("%s not set", tr.keyEnv)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			answer, err := ask(ctx, tr, key, doc)
			if tr.wire == wireRefused {
				if err == nil {
					t.Fatalf("%s accepted a PDF: %q", tr.label, answer)
				}
				t.Logf("refused: %v", err)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s answered %q", tr.label, answer)
			if !strings.Contains(strings.ToUpper(answer), secretWord) {
				t.Errorf("answer %q does not contain %s", answer, secretWord)
			}
		})
	}
}
