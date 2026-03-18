// Tutorial 20: LLM Capability Test
//
// Demonstrates: llm.NewClient, llm.Config, llm.Request, llm.Response,
//
//	llm.NewTextMessage, client.Complete, client.Provider, client.Model,
//	YAML-driven multi-model test matrices, markdown report generation.
//
// This tutorial shows how to systematically assess the reasoning and formatting
// capabilities of multiple LLM providers using the rho/llm library directly —
// no custom HTTP proxy required.
//
// Automatic skip logic:
//   - Cloud models: skipped when the required API key env var is empty.
//   - Ollama models: skipped when Ollama is not reachable or the model has not
//     been pulled (checked once at test start via GET /api/tags).
//
// Run:
//
//	go test -v -timeout 120m ./...
//
// Reports land in testdata/RESULTS_<timestamp>.md.
// Configure which models to test in config.yaml.
// Add or edit test prompts in tests.yaml.
package capability_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider" // register all provider adapters
	"gopkg.in/yaml.v3"
)

// =============================================================================
// Configuration
// =============================================================================

// Config is loaded from config.yaml.
type Config struct {
	ReportDir string        `yaml:"report_dir"`
	Models    []ModelConfig `yaml:"models"`
}

// ModelConfig describes one LLM client to test.
type ModelConfig struct {
	Provider       string `yaml:"provider"`
	Model          string `yaml:"model"`
	APIKeyEnv      string `yaml:"api_key_env"`
	BaseURL        string `yaml:"base_url"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

func loadConfig(t *testing.T) Config {
	t.Helper()
	b, err := os.ReadFile("config.yaml")
	if err != nil {
		b, err = os.ReadFile(filepath.Join("20_capability_test", "config.yaml"))
		if err != nil {
			t.Fatalf("Failed to load config.yaml: %v", err)
		}
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("Failed to parse config.yaml: %v", err)
	}
	return cfg
}

// =============================================================================
// Test Matrix (loaded from tests.yaml)
// =============================================================================

// Difficulty is a 1–5 scale: Easy → Master.
type Difficulty int

const (
	Level1_Easy   Difficulty = 1
	Level2_Medium Difficulty = 2
	Level3_Hard   Difficulty = 3
	Level4_Expert Difficulty = 4
	Level5_Master Difficulty = 5
)

// TestCase is one prompt + validator triple (EN / DE / ES).
type TestCase struct {
	ID         string
	Difficulty Difficulty
	Category   string
	PromptEN   string
	PromptDE   string
	PromptES   string
	// Validator returns true when the response meets the acceptance criteria.
	Validator func(response string) bool
}

// testCaseDef is the raw YAML shape before building closures.
type testCaseDef struct {
	ID            string   `yaml:"id"`
	Difficulty    int      `yaml:"difficulty"`
	Category      string   `yaml:"category"`
	PromptEN      string   `yaml:"prompt_en"`
	PromptDE      string   `yaml:"prompt_de"`
	PromptES      string   `yaml:"prompt_es"`
	ValidatorType string   `yaml:"validator_type"`
	Expected      []string `yaml:"expected"`
	NotExpected   []string `yaml:"not_expected"`
}

type testsConfig struct {
	Tests []testCaseDef `yaml:"tests"`
}

func loadTestMatrix(t *testing.T) []TestCase {
	t.Helper()
	b, err := os.ReadFile("tests.yaml")
	if err != nil {
		b, err = os.ReadFile(filepath.Join("20_capability_test", "tests.yaml"))
		if err != nil {
			t.Fatalf("Failed to load tests.yaml: %v", err)
		}
	}

	var cfg testsConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("Failed to parse tests.yaml: %v", err)
	}

	var testMatrix []TestCase

	for _, def := range cfg.Tests {
		tc := TestCase{
			ID:         def.ID,
			Difficulty: Difficulty(def.Difficulty),
			Category:   def.Category,
			PromptEN:   def.PromptEN,
			PromptDE:   def.PromptDE,
			PromptES:   def.PromptES,
		}

		// Capture slices before building closures.
		expected := make([]string, len(def.Expected))
		copy(expected, def.Expected)
		notExpected := make([]string, len(def.NotExpected))
		copy(notExpected, def.NotExpected)

		switch def.ValidatorType {
		case "json":
			tc.Validator = func(resp string) bool {
				text := strings.TrimSpace(resp)
				if strings.HasPrefix(text, "```json") {
					text = strings.TrimPrefix(text, "```json")
					text = strings.TrimSuffix(text, "```")
					text = strings.TrimSpace(text)
				}
				var j map[string]any
				return json.Unmarshal([]byte(text), &j) == nil
			}
		case "contains_all":
			tc.Validator = func(resp string) bool {
				for _, exp := range expected {
					if !containsAny(resp, exp) {
						return false
					}
				}
				if len(notExpected) > 0 && containsAny(resp, notExpected...) {
					return false
				}
				return true
			}
		default: // "contains_any"
			tc.Validator = func(resp string) bool {
				if len(expected) > 0 && !containsAny(resp, expected...) {
					return false
				}
				if len(notExpected) > 0 && containsAny(resp, notExpected...) {
					return false
				}
				return true
			}
		}

		testMatrix = append(testMatrix, tc)
	}
	return testMatrix
}

// =============================================================================
// Ollama availability check
// =============================================================================

// ollamaTagsResponse matches GET http://localhost:11434/api/tags.
type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// fetchOllamaModels queries the local Ollama daemon and returns the set of
// pulled model names. Returns nil (not an error) when Ollama is not running.
func fetchOllamaModels() map[string]bool {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil // Ollama not reachable
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var tags ollamaTagsResponse
	if err := json.Unmarshal(body, &tags); err != nil {
		return nil
	}

	available := make(map[string]bool, len(tags.Models))
	for _, m := range tags.Models {
		available[m.Name] = true
	}
	return available
}

// =============================================================================
// Test Suite
// =============================================================================

// TestStatus represents the outcome of a test.
type TestStatus string

const (
	StatusPass  TestStatus = "Pass"
	StatusFail  TestStatus = "Fail"
	StatusError TestStatus = "Error"
)

// TestResult records the outcome for one (model, test-case, language) triple.
type TestResult struct {
	Model      string
	Provider   string
	TestCaseID string
	Language   string
	Status     TestStatus
	Response   string
	Duration   time.Duration
}

func TestLLMCapabilities(t *testing.T) {
	testConfig := loadConfig(t)
	testMatrix := loadTestMatrix(t)

	if len(testConfig.Models) == 0 {
		t.Skip("No models configured in config.yaml. Add at least one entry under 'models:'.")
	}

	// Fetch available Ollama models once for the whole test run.
	ollamaModels := fetchOllamaModels()
	if ollamaModels == nil {
		t.Log("Ollama not reachable — all ollama models will be skipped.")
	} else {
		t.Logf("Ollama available: %d models pulled locally.", len(ollamaModels))
	}

	t.Logf("Running capability matrix: %d configured models × %d tests × 3 languages",
		len(testConfig.Models), len(testMatrix))

	var mu sync.Mutex
	var results []TestResult
	languages := []string{"EN", "DE", "ES"}

	// Register cleanup to generate the report after all parallel subtests finish.
	t.Cleanup(func() {
		mu.Lock()
		finalResults := make([]TestResult, len(results))
		copy(finalResults, results)
		mu.Unlock()

		if len(finalResults) == 0 {
			t.Log("No models ran (all skipped). Check API keys and Ollama status.")
			return
		}

		// Write a timestamped markdown report.
		timestamp := time.Now().Format("20060102_150405")
		reportFile := fmt.Sprintf("RESULTS_%s.md", timestamp)
		t.Logf("Writing report to %s/%s", testConfig.ReportDir, reportFile)
		if err := generateMarkdownReport(testConfig.ReportDir, reportFile, testMatrix, finalResults); err != nil {
			t.Errorf("Failed to write report: %v", err)
		}
	})

	// Group models by provider so each provider runs in parallel while
	// models within the same provider run sequentially (avoids rate-limit
	// storms on a single API key).
	providerGroups := make(map[string][]ModelConfig)
	for _, modelCfg := range testConfig.Models {
		providerGroups[modelCfg.Provider] = append(providerGroups[modelCfg.Provider], modelCfg)
	}

	for providerName, models := range providerGroups {
		t.Run(providerName, func(t *testing.T) {
			t.Parallel()

			for _, modelCfg := range models {
				label := modelCfg.Provider + "/" + modelCfg.Model

				// --- Skip check 1: API key required but not set ---
				if modelCfg.APIKeyEnv != "" && os.Getenv(modelCfg.APIKeyEnv) == "" {
					t.Logf("SKIP %s: env var %s is not set", label, modelCfg.APIKeyEnv)
					continue
				}

				// --- Skip check 2: Ollama model not pulled ---
				if modelCfg.Provider == "ollama" {
					if ollamaModels == nil {
						t.Logf("SKIP %s: Ollama is not reachable", label)
						continue
					}
					if !ollamaModels[modelCfg.Model] {
						t.Logf("SKIP %s: model not found in local Ollama (run: ollama pull %s)",
							label, modelCfg.Model)
						continue
					}
				}

				// --- Build the rho/llm client ---
				apiKey := ""
				if modelCfg.APIKeyEnv != "" {
					apiKey = os.Getenv(modelCfg.APIKeyEnv)
				}

				timeout := 120 * time.Second
				if modelCfg.TimeoutSeconds > 0 {
					timeout = time.Duration(modelCfg.TimeoutSeconds) * time.Second
				}

				cfg := llm.Config{
					Provider:  modelCfg.Provider,
					Model:     modelCfg.Model,
					APIKey:    apiKey,
					MaxTokens: 2048, // generous budget for thinking models
					Timeout:   timeout,
				}
				if modelCfg.BaseURL != "" {
					cfg.BaseURL = modelCfg.BaseURL
				}

				client, err := llm.NewClient(cfg)
				if err != nil {
					t.Logf("SKIP %s: cannot create client: %v", label, err)
					continue
				}

				// Use the resolved IDs from the client itself (handles aliases).
				resolvedModel := client.Model()
				resolvedProvider := client.Provider()

				t.Run(resolvedModel, func(t *testing.T) {
					defer client.Close()

					for _, tc := range testMatrix {
						for _, lang := range languages {
							testName := fmt.Sprintf("%s/%s/%s", tc.Category, tc.ID, lang)

							t.Run(testName, func(t *testing.T) {
								var prompt string
								switch lang {
								case "EN":
									prompt = tc.PromptEN
								case "DE":
									prompt = tc.PromptDE
								case "ES":
									prompt = tc.PromptES
								}

								start := time.Now()
								respText, err := complete(t.Context(), client, timeout, prompt)
								duration := time.Since(start)

								if err != nil {
									t.Logf("ERROR %s: %v", resolvedModel, err)
									mu.Lock()
									results = append(results, TestResult{
										Model: resolvedModel, Provider: resolvedProvider,
										TestCaseID: tc.ID, Language: lang,
										Status:   StatusError,
										Response: fmt.Sprintf("ERROR: %v", err),
										Duration: duration,
									})
									mu.Unlock()
									return
								}

								passed := tc.Validator(respText)
								status := StatusPass
								if !passed {
									status = StatusFail
									t.Logf("FAIL  %s — expected condition not met. Output: %q", resolvedModel, respText)
								} else {
									t.Logf("PASS  %s", resolvedModel)
								}

								mu.Lock()
								results = append(results, TestResult{
									Model: resolvedModel, Provider: resolvedProvider,
									TestCaseID: tc.ID, Language: lang,
									Status:   status,
									Response: respText,
									Duration: duration,
								})
								mu.Unlock()
							})
						}
					}
				})
			}
		})
	}
}

// =============================================================================
// Helpers
// =============================================================================

// complete sends a single user message to the client and returns the text response.
// parent should be t.Context() so that overall test cancellation cascades to HTTP.
func complete(parent context.Context, client llm.Client, timeout time.Duration, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	req := llm.Request{
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, prompt),
		},
		MaxTokens: 2048,
	}

	resp, err := client.Complete(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// containsAny reports whether s (lowercased, punctuation-stripped) contains any
// of the given substrings (also lowercased).
var stripPunct = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)

func normalize(s string) string {
	return stripPunct.ReplaceAllString(strings.ToLower(s), "")
}

func containsAny(s string, checks ...string) bool {
	s = normalize(s)
	for _, c := range checks {
		if strings.Contains(s, normalize(c)) {
			return true
		}
	}
	return false
}

// =============================================================================
// Markdown Report
// =============================================================================

func generateMarkdownReport(dir, filename string, testMatrix []TestCase, results []TestResult) error {
	var sb strings.Builder

	sb.WriteString("# LLM Capability Regression Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated on: `%s`\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Test matrix summary table
	sb.WriteString("## Test Matrix Summary\n\n")
	sb.WriteString("| ID | Category | Difficulty | Prompt (EN) |\n")
	sb.WriteString("|---|---|---|---|\n")
	for _, tc := range testMatrix {
		sb.WriteString(fmt.Sprintf("| `%s` | %s | Level %d | %q |\n",
			tc.ID, tc.Category, tc.Difficulty, tc.PromptEN))
	}
	sb.WriteString("\n")

	// Collect distinct models in result order
	type modelKey struct{ model, provider string }
	seen := map[modelKey]bool{}
	var models []modelKey
	for _, r := range results {
		k := modelKey{r.Model, r.Provider}
		if !seen[k] {
			seen[k] = true
			models = append(models, k)
		}
	}

	// Group results by model
	grouped := map[string][]TestResult{}
	for _, r := range results {
		grouped[r.Model] = append(grouped[r.Model], r)
	}

	// Build scoreboard
	type score struct {
		model    string
		provider string
		passed   int
		failed   int
		errors   int
		total    int
		avgDur   time.Duration
	}
	var scores []score
	for _, m := range models {
		mRes := grouped[m.model]
		if len(mRes) == 0 {
			continue
		}
		passed := 0
		failed := 0
		errors := 0
		var totalDur time.Duration
		for _, r := range mRes {
			switch r.Status {
			case StatusPass:
				passed++
			case StatusFail:
				failed++
			case StatusError:
				errors++
			}
			totalDur += r.Duration
		}
		scores = append(scores, score{
			model:    m.model,
			provider: m.provider,
			passed:   passed,
			failed:   failed,
			errors:   errors,
			total:    len(mRes),
			avgDur:   totalDur / time.Duration(len(mRes)),
		})
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].passed != scores[j].passed {
			return scores[i].passed > scores[j].passed
		}
		return scores[i].model < scores[j].model
	})

	sb.WriteString("## Scoreboard\n\n")
	sb.WriteString("| Model | Provider | Pass Rate | Passed | Failed | Errors | Avg Time |\n")
	sb.WriteString("|---|---|---|---|---|---|---|\n")
	for _, s := range scores {
		rate := float64(s.passed) / float64(s.total) * 100
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %.1f%% | %d | %d | %d | %dms |\n",
			s.model, s.provider, rate, s.passed, s.failed, s.errors, s.avgDur.Milliseconds()))
	}
	sb.WriteString("\n")

	// Per-test detail grid
	sb.WriteString("## Detailed Results\n\n")
	for _, tc := range testMatrix {
		sb.WriteString(fmt.Sprintf("### %s (Level %d)\n\n", tc.ID, tc.Difficulty))
		sb.WriteString("| Model | EN | DE | ES |\n")
		sb.WriteString("|---|---|---|---|\n")
		for _, s := range scores {
			en := findResult(grouped[s.model], tc.ID, "EN")
			de := findResult(grouped[s.model], tc.ID, "DE")
			es := findResult(grouped[s.model], tc.ID, "ES")
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n",
				s.model, formatStatus(en.Status), formatStatus(de.Status), formatStatus(es.Status)))
		}
		sb.WriteString("\n")
	}

	// Raw outputs for failed tests
	sb.WriteString("## Raw Outputs (Failed Tests)\n\n")
	sb.WriteString("<details><summary>Click to expand</summary>\n\n")
	for _, m := range models {
		for _, r := range grouped[m.model] {
			if r.Status != StatusPass {
				sb.WriteString(fmt.Sprintf("#### `%s` — %s (%s) [%s]\n", r.Model, r.TestCaseID, r.Language, r.Status))
				sb.WriteString("```\n")
				sb.WriteString(strings.TrimSpace(r.Response))
				sb.WriteString("\n```\n\n")
			}
		}
	}
	sb.WriteString("</details>\n")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), []byte(sb.String()), 0644)
}

func findResult(results []TestResult, tcID, lang string) TestResult {
	for _, r := range results {
		if r.TestCaseID == tcID && r.Language == lang {
			return r
		}
	}
	return TestResult{}
}

func formatStatus(s TestStatus) string {
	switch s {
	case StatusPass:
		return "✅"
	case StatusFail:
		return "❌"
	case StatusError:
		return "⚠️"
	default:
		return "➖"
	}
}
