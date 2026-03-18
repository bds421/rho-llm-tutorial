// Tutorial 21: Cloud-CTL Tool Use Benchmark
//
// Demonstrates: llm.Tool, llm.ToolCall, llm.NewAssistantMessage,
//               llm.NewToolResultMessage, agentic tool-use loop,
//               YAML-driven multi-model test matrix, markdown report generation.
//
// Runs every tool call against the real `cl` binary (cloud-ctl CLI).
// Requires: `cl` in PATH + valid authentication (CLOUD_ACCOUNT / OAuth).
// Skips automatically if `cl` is not available or auth fails.
//
// Run:
//
//	go test -v -timeout 60m ./...
//
// Reports land in testdata/RESULTS_TOOL_USE_<timestamp>.md.
package cloud_ctl_tool_use

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/bds421/rho-llm"
	_ "github.com/bds421/rho-llm/provider" // register all provider adapters
	"gopkg.in/yaml.v3"
)

// =============================================================================
// Configuration
// =============================================================================

type Config struct {
	ReportDir string        `yaml:"report_dir"`
	Models    []ModelConfig `yaml:"models"`
}

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
		b, err = os.ReadFile(filepath.Join("21_cloud_ctl_tool_use", "config.yaml"))
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

type Difficulty int

const (
	Level1_SingleTool       Difficulty = 1
	Level2_Arguments        Difficulty = 2
	Level3_MultiTool        Difficulty = 3
	Level4_ErrorRecovery    Difficulty = 4
	Level5_ComplexReasoning Difficulty = 5
)

type TestCase struct {
	ID                string
	Difficulty        Difficulty
	Category          string
	Prompt            string
	ExpectedTools     []string
	AcceptableTools   []string            // alternative tools also considered correct
	ExpectedArgs      map[string][]string // tool name -> expected argument keywords
	ValidatorType     string
	ErrorKeywords     []string // for handles_error
	EmptyKeywords     []string // for handles_empty
	SynthesisKeywords []string // for all_tools_called_and_coherent
}

type testCaseDef struct {
	ID                string              `yaml:"id"`
	Difficulty        int                 `yaml:"difficulty"`
	Category          string              `yaml:"category"`
	Prompt            string              `yaml:"prompt"`
	ExpectedTools     []string            `yaml:"expected_tools"`
	AcceptableTools   []string            `yaml:"acceptable_tools"`
	ExpectedArgs      map[string][]string `yaml:"expected_args"`
	ValidatorType     string              `yaml:"validator_type"`
	ErrorKeywords     []string            `yaml:"error_keywords"`
	EmptyKeywords     []string            `yaml:"empty_keywords"`
	SynthesisKeywords []string            `yaml:"synthesis_keywords"`
}

type testsConfig struct {
	Tests []testCaseDef `yaml:"tests"`
}

func loadTestMatrix(t *testing.T) []TestCase {
	t.Helper()
	b, err := os.ReadFile("tests.yaml")
	if err != nil {
		b, err = os.ReadFile(filepath.Join("21_cloud_ctl_tool_use", "tests.yaml"))
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
			ID:                def.ID,
			Difficulty:        Difficulty(def.Difficulty),
			Category:          def.Category,
			Prompt:            def.Prompt,
			ExpectedTools:     def.ExpectedTools,
			AcceptableTools:   def.AcceptableTools,
			ExpectedArgs:      def.ExpectedArgs,
			ValidatorType:     def.ValidatorType,
			ErrorKeywords:     def.ErrorKeywords,
			EmptyKeywords:     def.EmptyKeywords,
			SynthesisKeywords: def.SynthesisKeywords,
		}
		testMatrix = append(testMatrix, tc)
	}
	return testMatrix
}

// =============================================================================
// cl binary availability check
// =============================================================================

// checkCLAvailable verifies the cl binary is in PATH and authenticated.
// Returns an error message if not available, empty string if OK.
func checkCLAvailable() string {
	if _, err := exec.LookPath("cl"); err != nil {
		return "cl binary not found in PATH (build with: go build -o cl ./cmd/cl/ in rho/cloud-ctl)"
	}
	// Smoke test: list calendars (lightweight, read-only).
	cmd := exec.Command("cl", "calendar", "calendars", "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("cl auth check failed: %s: %s", err, strings.TrimSpace(string(out)))
	}
	return ""
}

// =============================================================================
// Ollama availability check (same pattern as tutorial 20)
// =============================================================================

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func fetchOllamaModels() map[string]bool {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil
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
// Test Result Types
// =============================================================================

type TestStatus string

const (
	StatusPass  TestStatus = "Pass"
	StatusFail  TestStatus = "Fail"
	StatusError TestStatus = "Error"
)

type TestResult struct {
	Model       string
	Provider    string
	TestCaseID  string
	Status      TestStatus
	Response    string        // final text response
	Duration    time.Duration
	Rounds      int           // number of agentic loop iterations
	ToolsCalled []string      // tool names invoked
	ToolTraces  []ToolTrace   // full trace of tool calls
	ToolCorrect bool          // expected tool(s) called
	ArgsCorrect bool          // expected arguments present
}

// =============================================================================
// Agentic Loop
// =============================================================================

const maxRounds = 10

// runAgenticLoop executes the tool-use agentic loop for a single test case.
func runAgenticLoop(ctx context.Context, client llm.Client, tc TestCase) (string, []ToolTrace, int, error) {
	tools := CloudTools()

	req := llm.Request{
		System: SystemPrompt(),
		Messages: []llm.Message{
			llm.NewTextMessage(llm.RoleUser, tc.Prompt),
		},
		Tools:     tools,
		MaxTokens: 4096,
	}

	resp, err := client.Complete(ctx, req)
	if err != nil {
		return "", nil, 0, err
	}

	var allTraces []ToolTrace
	rounds := 0

	for resp.StopReason == "tool_use" && rounds < maxRounds {
		rounds++

		var results []llm.Message
		for _, tc := range resp.ToolCalls {
			output, isError := ExecuteTool(tc.Name, tc.Input)

			allTraces = append(allTraces, ToolTrace{
				Name:    tc.Name,
				Input:   tc.Input,
				Output:  output,
				IsError: isError,
			})

			results = append(results, llm.NewToolResultMessage(tc.ID, output, isError))
		}

		// Preserve tool_use blocks in conversation history (not just text).
		req.Messages = append(req.Messages, llm.NewAssistantMessage(resp))
		req.Messages = append(req.Messages, results...)

		resp, err = client.Complete(ctx, req)
		if err != nil {
			return "", allTraces, rounds, err
		}
	}

	return resp.Content, allTraces, rounds, nil
}

// =============================================================================
// Validators
// =============================================================================

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

// toolsCalled extracts distinct tool names from traces.
func toolsCalled(traces []ToolTrace) []string {
	seen := map[string]bool{}
	var names []string
	for _, t := range traces {
		if !seen[t.Name] {
			seen[t.Name] = true
			names = append(names, t.Name)
		}
	}
	return names
}

// allValidTools returns the union of ExpectedTools and AcceptableTools.
func allValidTools(tc TestCase) map[string]bool {
	valid := make(map[string]bool)
	for _, t := range tc.ExpectedTools {
		valid[t] = true
	}
	for _, t := range tc.AcceptableTools {
		valid[t] = true
	}
	return valid
}

// validateToolCalled checks that at least one expected or acceptable tool was called.
func validateToolCalled(tc TestCase, traces []ToolTrace) bool {
	valid := allValidTools(tc)
	for _, t := range traces {
		if valid[t.Name] {
			return true
		}
	}
	return false
}

// validateToolCalledWithArg checks tool selection + argument keywords.
func validateToolCalledWithArg(tc TestCase, traces []ToolTrace) (toolOK, argsOK bool) {
	toolOK = validateToolCalled(tc, traces)
	if !toolOK {
		return false, false
	}

	if len(tc.ExpectedArgs) == 0 {
		return true, true
	}

	argsOK = true
	for toolName, expectedKWs := range tc.ExpectedArgs {
		found := false
		for _, trace := range traces {
			if trace.Name != toolName {
				continue
			}
			inputJSON, _ := json.Marshal(trace.Input)
			inputStr := strings.ToLower(string(inputJSON))
			allPresent := true
			for _, kw := range expectedKWs {
				if !strings.Contains(inputStr, strings.ToLower(kw)) {
					allPresent = false
					break
				}
			}
			if allPresent {
				found = true
				break
			}
		}
		if !found {
			argsOK = false
		}
	}
	return toolOK, argsOK
}

// validateAllToolsCalled checks that ALL expected tools were called (any order).
// Also passes if acceptable alternatives cover the expected set.
func validateAllToolsCalled(tc TestCase, traces []ToolTrace) bool {
	calledSet := map[string]bool{}
	for _, t := range traces {
		calledSet[t.Name] = true
	}
	for _, exp := range tc.ExpectedTools {
		if !calledSet[exp] {
			// Check if an acceptable alternative was called instead.
			if len(tc.AcceptableTools) > 0 {
				hasAlternative := false
				for _, alt := range tc.AcceptableTools {
					if calledSet[alt] {
						hasAlternative = true
						break
					}
				}
				if !hasAlternative {
					return false
				}
			} else {
				return false
			}
		}
	}
	return true
}

// validateHandlesError checks that an error tool result was received and the
// model acknowledges it without hallucinating data.
func validateHandlesError(tc TestCase, traces []ToolTrace, response string) bool {
	if !validateToolCalled(tc, traces) {
		return false
	}
	hasError := false
	for _, t := range traces {
		if t.IsError {
			hasError = true
			break
		}
	}
	if !hasError {
		return false
	}
	return containsAny(response, tc.ErrorKeywords...)
}

// validateHandlesEmpty checks that empty/no results are acknowledged.
func validateHandlesEmpty(tc TestCase, traces []ToolTrace, response string) bool {
	if !validateToolCalled(tc, traces) {
		return false
	}
	return containsAny(response, tc.EmptyKeywords...)
}

// validateAllToolsCalledAndCoherent checks all tools called + synthesis keywords
// present in the final response.
func validateAllToolsCalledAndCoherent(tc TestCase, traces []ToolTrace, response string) bool {
	if !validateAllToolsCalled(tc, traces) {
		return false
	}
	for _, kw := range tc.SynthesisKeywords {
		if !containsAny(response, kw) {
			return false
		}
	}
	return true
}

// =============================================================================
// Main Test
// =============================================================================

func TestCloudCtlToolUse(t *testing.T) {
	// Gate: cl binary must be available and authenticated.
	if msg := checkCLAvailable(); msg != "" {
		t.Skipf("SKIP: %s", msg)
	}
	t.Log("cl binary available and authenticated")

	testConfig := loadConfig(t)
	testMatrix := loadTestMatrix(t)

	if len(testConfig.Models) == 0 {
		t.Skip("No models configured in config.yaml.")
	}

	ollamaModels := fetchOllamaModels()
	if ollamaModels == nil {
		t.Log("Ollama not reachable -- all ollama models will be skipped.")
	} else {
		t.Logf("Ollama available: %d models pulled locally.", len(ollamaModels))
	}

	t.Logf("Running tool-use matrix: %d configured models x %d tests",
		len(testConfig.Models), len(testMatrix))

	var results []TestResult

	for _, modelCfg := range testConfig.Models {
		modelCfg := modelCfg
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

		timeout := 180 * time.Second
		if modelCfg.TimeoutSeconds > 0 {
			timeout = time.Duration(modelCfg.TimeoutSeconds) * time.Second
		}

		cfg := llm.Config{
			Provider:  modelCfg.Provider,
			Model:     modelCfg.Model,
			APIKey:    apiKey,
			MaxTokens: 4096,
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

		resolvedModel := client.Model()
		resolvedProvider := client.Provider()

		t.Run(resolvedProvider+"/"+resolvedModel, func(t *testing.T) {
			defer client.Close()

			for _, tc := range testMatrix {
				tc := tc
				testName := fmt.Sprintf("%s/%s", tc.Category, tc.ID)

				t.Run(testName, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(t.Context(), timeout)
					defer cancel()

					start := time.Now()
					response, traces, rounds, err := runAgenticLoop(ctx, client, tc)
					duration := time.Since(start)

					if err != nil {
						t.Logf("ERROR %s: %v", resolvedModel, err)
						results = append(results, TestResult{
							Model: resolvedModel, Provider: resolvedProvider,
							TestCaseID: tc.ID, Status: StatusError,
							Response: fmt.Sprintf("ERROR: %v", err),
							Duration: duration, Rounds: rounds,
							ToolTraces: traces,
						})
						return
					}

					called := toolsCalled(traces)
					var toolCorrect, argsCorrect, passed bool

					switch tc.ValidatorType {
					case "tool_called":
						toolCorrect = validateToolCalled(tc, traces)
						argsCorrect = true
						passed = toolCorrect

					case "tool_called_with_arg":
						toolCorrect, argsCorrect = validateToolCalledWithArg(tc, traces)
						passed = toolCorrect && argsCorrect

					case "all_tools_called":
						toolCorrect = validateAllToolsCalled(tc, traces)
						argsCorrect = true
						passed = toolCorrect

					case "handles_error":
						toolCorrect = validateToolCalled(tc, traces)
						argsCorrect = true
						passed = validateHandlesError(tc, traces, response)

					case "handles_empty":
						toolCorrect = validateToolCalled(tc, traces)
						argsCorrect = true
						passed = validateHandlesEmpty(tc, traces, response)

					case "all_tools_called_and_coherent":
						toolCorrect = validateAllToolsCalled(tc, traces)
						argsCorrect = true
						passed = validateAllToolsCalledAndCoherent(tc, traces, response)
					}

					status := StatusPass
					if !passed {
						status = StatusFail
						t.Logf("FAIL  %s -- tools called: %v, expected: %v, toolOK=%v, argsOK=%v",
							resolvedModel, called, tc.ExpectedTools, toolCorrect, argsCorrect)
					} else {
						t.Logf("PASS  %s (%d rounds, tools: %v)", resolvedModel, rounds, called)
					}

					results = append(results, TestResult{
						Model: resolvedModel, Provider: resolvedProvider,
						TestCaseID: tc.ID, Status: status,
						Response: response, Duration: duration,
						Rounds: rounds, ToolsCalled: called,
						ToolTraces:  traces,
						ToolCorrect: toolCorrect, ArgsCorrect: argsCorrect,
					})
				})
			}
		})
	}

	if len(results) == 0 {
		t.Skip("No models ran (all skipped). Check API keys and Ollama status.")
	}

	timestamp := time.Now().Format("20060102_150405")
	reportFile := fmt.Sprintf("RESULTS_TOOL_USE_%s.md", timestamp)
	t.Logf("Writing report to %s/%s", testConfig.ReportDir, reportFile)
	if err := generateReport(testConfig.ReportDir, reportFile, testMatrix, results); err != nil {
		t.Errorf("Failed to write report: %v", err)
	}
}

// =============================================================================
// Markdown Report
// =============================================================================

func generateReport(dir, filename string, testMatrix []TestCase, results []TestResult) error {
	var sb strings.Builder

	sb.WriteString("# Cloud-CTL Tool Use Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: `%s`\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Tool definitions summary
	sb.WriteString("## Tool Definitions\n\n")
	sb.WriteString("| Tool | CLI Command | Description |\n")
	sb.WriteString("|---|---|---|\n")
	for _, tool := range CloudTools() {
		sb.WriteString(fmt.Sprintf("| `%s` | `cl %s` | %s |\n",
			tool.Name, toolToCLI(tool.Name), truncate(tool.Description, 60)))
	}
	sb.WriteString("\n")

	// Test scenario summary
	sb.WriteString("## Test Scenarios\n\n")
	sb.WriteString("| ID | Level | Category | Prompt | Expected Tools |\n")
	sb.WriteString("|---|---|---|---|---|\n")
	for _, tc := range testMatrix {
		sb.WriteString(fmt.Sprintf("| `%s` | %d | %s | %s | %s |\n",
			tc.ID, tc.Difficulty, tc.Category,
			truncate(tc.Prompt, 50),
			strings.Join(tc.ExpectedTools, ", ")))
	}
	sb.WriteString("\n")

	// Collect distinct models
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

	grouped := map[string][]TestResult{}
	for _, r := range results {
		grouped[r.Model] = append(grouped[r.Model], r)
	}

	// Scoreboard
	type score struct {
		model        string
		provider     string
		passed       int
		failed       int
		errors       int
		total        int
		toolAccuracy int
		argAccuracy  int
		avgRounds    float64
		avgDur       time.Duration
	}
	var scores []score
	for _, m := range models {
		mRes := grouped[m.model]
		if len(mRes) == 0 {
			continue
		}
		var s score
		s.model = m.model
		s.provider = m.provider
		s.total = len(mRes)
		var totalRounds int
		var totalDur time.Duration
		for _, r := range mRes {
			switch r.Status {
			case StatusPass:
				s.passed++
			case StatusFail:
				s.failed++
			case StatusError:
				s.errors++
			}
			if r.ToolCorrect {
				s.toolAccuracy++
			}
			if r.ArgsCorrect {
				s.argAccuracy++
			}
			totalRounds += r.Rounds
			totalDur += r.Duration
		}
		s.avgRounds = float64(totalRounds) / float64(len(mRes))
		s.avgDur = totalDur / time.Duration(len(mRes))
		scores = append(scores, s)
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].passed != scores[j].passed {
			return scores[i].passed > scores[j].passed
		}
		return scores[i].model < scores[j].model
	})

	sb.WriteString("## Scoreboard\n\n")
	sb.WriteString("| Model | Provider | Pass Rate | Tool Acc | Arg Acc | Avg Rounds | Avg Latency | Pass | Fail | Err |\n")
	sb.WriteString("|---|---|---|---|---|---|---|---|---|---|\n")
	for _, s := range scores {
		passRate := float64(s.passed) / float64(s.total) * 100
		toolAcc := float64(s.toolAccuracy) / float64(s.total) * 100
		argAcc := float64(s.argAccuracy) / float64(s.total) * 100
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %.0f%% | %.0f%% | %.0f%% | %.1f | %dms | %d | %d | %d |\n",
			s.model, s.provider, passRate, toolAcc, argAcc,
			s.avgRounds, s.avgDur.Milliseconds(),
			s.passed, s.failed, s.errors))
	}
	sb.WriteString("\n")

	// Per-test detail grid
	sb.WriteString("## Detailed Results\n\n")
	for _, tc := range testMatrix {
		sb.WriteString(fmt.Sprintf("### %s (Level %d - %s)\n\n", tc.ID, tc.Difficulty, tc.Category))
		sb.WriteString("| Model | Status | Tools Called | Rounds | Latency |\n")
		sb.WriteString("|---|---|---|---|---|\n")
		for _, s := range scores {
			r := findResult(grouped[s.model], tc.ID)
			if r.TestCaseID == "" {
				sb.WriteString(fmt.Sprintf("| `%s` | - | - | - | - |\n", s.model))
				continue
			}
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %d | %dms |\n",
				s.model, formatStatus(r.Status),
				strings.Join(r.ToolsCalled, ", "),
				r.Rounds, r.Duration.Milliseconds()))
		}
		sb.WriteString("\n")
	}

	// Raw tool call traces for failed tests
	sb.WriteString("## Raw Tool Call Traces (Failed/Error Tests)\n\n")
	sb.WriteString("<details><summary>Click to expand</summary>\n\n")
	for _, m := range models {
		for _, r := range grouped[m.model] {
			if r.Status == StatusPass {
				continue
			}
			sb.WriteString(fmt.Sprintf("#### `%s` -- %s [%s]\n\n", r.Model, r.TestCaseID, r.Status))
			if len(r.ToolTraces) > 0 {
				sb.WriteString("**Tool Calls:**\n```json\n")
				traceJSON, _ := json.MarshalIndent(r.ToolTraces, "", "  ")
				sb.WriteString(string(traceJSON))
				sb.WriteString("\n```\n\n")
			}
			sb.WriteString("**Final Response:**\n```\n")
			sb.WriteString(strings.TrimSpace(r.Response))
			sb.WriteString("\n```\n\n")
		}
	}
	sb.WriteString("</details>\n")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), []byte(sb.String()), 0644)
}

// =============================================================================
// Helpers
// =============================================================================

func findResult(results []TestResult, tcID string) TestResult {
	for _, r := range results {
		if r.TestCaseID == tcID {
			return r
		}
	}
	return TestResult{}
}

func formatStatus(s TestStatus) string {
	switch s {
	case StatusPass:
		return "PASS"
	case StatusFail:
		return "FAIL"
	case StatusError:
		return "ERR"
	default:
		return "-"
	}
}

func toolToCLI(name string) string {
	switch name {
	case "drive_ls":
		return "drive ls --json"
	case "drive_search":
		return "drive search Q --json"
	case "email_search":
		return "email search Q --json"
	case "calendar_list":
		return "calendar list --json"
	case "calendar_calendars":
		return "calendar calendars --json"
	case "sheets_read":
		return "sheets read ID --json"
	default:
		return name
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
