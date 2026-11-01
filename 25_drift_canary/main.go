// Tutorial 25: Drift Canary — detect a nerfed, quantized or rerouted model API
//
// Demonstrates: llm.NewClient, llm.Request (Temperature, ThinkingLevel,
//
//	multi-turn Messages), llm.Response metadata (Model, OutputTokens,
//	ThinkingTokens, StopReason), seeded task generation, exact checkers,
//	JSONL run logs, and statistical run-to-run comparison.
//
// The idea: famous IQ puzzles do not reveal a degraded model, because every
// model has memorized them. This tool generates fresh tasks from a seed,
// grades them exactly, records the response metadata, and compares runs with
// paired statistics, so a real drop shows up as a significant change instead
// of a feeling.
//
// Task groups:
//   - intent     — does the model understand terse, typo-laden, mixed-language
//     user messages? (distilled from real agent sessions where the
//     model misread the user)
//   - agentic    — counting from records, current vs superseded facts, noticing
//     finished work, reading tool flags, choosing the cheap path
//   - precision  — multi-digit multiplication, state tracking, exact recall,
//     exact string reversal (sensitive to quantization)
//
// Run:
//
//	go run . run  -config config.yaml -set fixed          # baseline / repeat runs
//	go run . run  -set fresh -count 2                      # new unseen tasks
//	go run . compare -base runs/A.jsonl -cand runs/B.jsonl # did it get worse?
//	go run . summary runs/A.jsonl
//	go run . show -family intent -level 3 -idx 0           # inspect a task
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
	OutDir string         `yaml:"out_dir"`
	Count  int            `yaml:"count"`  // tasks per family weight unit
	Levels map[string]int `yaml:"levels"` // difficulty per family (default 3)
	Models []ModelConfig  `yaml:"models"`
}

// ModelConfig describes one model endpoint. Pin everything that could change
// the result, so that a difference between runs comes from the provider.
type ModelConfig struct {
	Provider       string   `yaml:"provider"`
	Model          string   `yaml:"model"`
	APIKeyEnv      string   `yaml:"api_key_env"`
	BaseURL        string   `yaml:"base_url"`
	Thinking       string   `yaml:"thinking"`    // "", low, medium, high
	Temperature    *float64 `yaml:"temperature"` // nil = provider default
	MaxTokens      int      `yaml:"max_tokens"`  // default 8192
	TimeoutSeconds int      `yaml:"timeout_seconds"`
	Concurrency    int      `yaml:"concurrency"` // parallel requests, default 4
	// Capabilities declares a model the rho/llm registry does not know yet,
	// e.g. [chat, stream, reasoning]. Leave empty for registered models.
	Capabilities []string `yaml:"capabilities"`
}

var capabilityNames = map[string]llm.Capability{
	"chat":        llm.CapabilityChat,
	"stream":      llm.CapabilityStream,
	"tools":       llm.CapabilityTools,
	"reasoning":   llm.CapabilityReasoning,
	"temperature": llm.CapabilityTemperature,
}

func capabilitySet(names []string) (llm.CapabilitySet, error) {
	var caps []llm.Capability
	for _, n := range names {
		c, ok := capabilityNames[n]
		if !ok {
			return 0, fmt.Errorf("unknown capability %q", n)
		}
		caps = append(caps, c)
	}
	return llm.Capabilities(caps...), nil
}

func loadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.OutDir == "" {
		cfg.OutDir = "runs"
	}
	if cfg.Count <= 0 {
		cfg.Count = 10
	}
	return cfg, nil
}

func (c Config) level(family string) int {
	if l, ok := c.Levels[family]; ok && l >= 1 && l <= 5 {
		return l
	}
	return 3
}

// =============================================================================
// Result record (one JSONL line per graded answer)
// =============================================================================

// Record is one graded model answer.
type Record struct {
	RunID          string   `json:"run_id"`
	Time           string   `json:"time"`
	Set            string   `json:"set"`
	Provider       string   `json:"provider"`
	Model          string   `json:"model"`
	ServedModel    string   `json:"served_model"` // what the API says it used
	Thinking       string   `json:"thinking,omitempty"`
	Temperature    *float64 `json:"temperature,omitempty"`
	Family         string   `json:"family"`
	Group          string   `json:"group"`
	Level          int      `json:"level"`
	TaskID         string   `json:"task_id"`
	Sample         int      `json:"sample"`
	Outcome        string   `json:"outcome"` // pass, fail, no_answer, truncated, refusal, error
	Extracted      string   `json:"extracted,omitempty"`
	Gold           string   `json:"gold"`
	Response       string   `json:"response,omitempty"`
	Error          string   `json:"error,omitempty"`
	LatencyMS      int64    `json:"latency_ms"`
	InputTokens    int      `json:"input_tokens"`
	OutputTokens   int      `json:"output_tokens"`
	ThinkingTokens int      `json:"thinking_tokens,omitempty"`
	ThinkingChars  int      `json:"thinking_chars,omitempty"`
	StopReason     string   `json:"stop_reason,omitempty"`
}

// Correct reports whether the record counts as a pass.
func (r Record) Correct() bool { return r.Outcome == "pass" }

// =============================================================================
// main
// =============================================================================

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = cmdRun(os.Args[2:])
	case "compare":
		err = cmdCompare(os.Args[2:])
	case "summary":
		err = cmdSummary(os.Args[2:])
	case "show":
		err = cmdShow(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  drift-canary run     [-config config.yaml] [-set fixed|fresh|<name>] [-count N] [-samples N] [-families a,b] [-models x,y]
  drift-canary compare -base runs/a.jsonl[,more.jsonl] -cand runs/b.jsonl[,more.jsonl] [-alpha 0.01]
  drift-canary summary runs/a.jsonl [more.jsonl ...]
  drift-canary show    [-family intent] [-level 3] [-idx 0] [-set fixed]`)
}

// =============================================================================
// run
// =============================================================================

// FixedSet is the seed set that never changes: identical tasks on every run,
// so two runs can be compared pair by pair.
const FixedSet = "fixed-v1"

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "model config")
	set := fs.String("set", "fixed", "task seed set: fixed, fresh, or any name")
	count := fs.Int("count", 0, "tasks per family weight unit (overrides config)")
	samples := fs.Int("samples", 1, "answers per task")
	famFilter := fs.String("families", "", "comma-separated families (default all)")
	modelFilter := fs.String("models", "", "comma-separated substrings of model IDs (default all)")
	fs.Parse(args)

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	if *count > 0 {
		cfg.Count = *count
	}
	setName := *set
	switch setName {
	case "fixed":
		setName = FixedSet
	case "fresh":
		setName = "fresh-" + time.Now().Format("20060102-150405")
	}

	tasks, err := buildTasks(cfg, setName, splitList(*famFilter))
	if err != nil {
		return err
	}

	runID := time.Now().Format("20060102-150405")
	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return err
	}
	outPath := filepath.Join(cfg.OutDir, runID+".jsonl")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	var wmu sync.Mutex
	write := func(rec Record) {
		b, _ := json.Marshal(rec)
		wmu.Lock()
		w.Write(append(b, '\n'))
		w.Flush()
		wmu.Unlock()
	}

	fmt.Fprintf(os.Stderr, "run %s: set %s, %d tasks × %d sample(s) per model → %s\n",
		runID, setName, len(tasks), *samples, outPath)

	var wg sync.WaitGroup
	for _, mc := range cfg.Models {
		if !matchAny(mc.Model, splitList(*modelFilter)) {
			continue
		}
		if mc.APIKeyEnv != "" && os.Getenv(mc.APIKeyEnv) == "" {
			fmt.Fprintf(os.Stderr, "SKIP %s/%s: %s not set\n", mc.Provider, mc.Model, mc.APIKeyEnv)
			continue
		}
		wg.Add(1)
		go func(mc ModelConfig) {
			defer wg.Done()
			if err := runModel(mc, runID, setName, tasks, *samples, write); err != nil {
				fmt.Fprintf(os.Stderr, "SKIP %s/%s: %v\n", mc.Provider, mc.Model, err)
			}
		}(mc)
	}
	wg.Wait()
	fmt.Fprintf(os.Stderr, "\nwrote %s\n\n", outPath)
	return summarize(os.Stdout, []string{outPath})
}

func buildTasks(cfg Config, set string, families []string) ([]Task, error) {
	var tasks []Task
	for _, fam := range Families {
		if len(families) > 0 && !contains(families, fam.Name) {
			continue
		}
		n := cfg.Count * fam.Weight
		for i := 0; i < n; i++ {
			t, err := Generate(set, fam.Name, cfg.level(fam.Name), i)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, t)
		}
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks selected")
	}
	return tasks, nil
}

func runModel(mc ModelConfig, runID, set string, tasks []Task, samples int, write func(Record)) error {
	timeout := 300 * time.Second
	if mc.TimeoutSeconds > 0 {
		timeout = time.Duration(mc.TimeoutSeconds) * time.Second
	}
	maxTokens := mc.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}
	conc := mc.Concurrency
	if conc <= 0 {
		conc = 4
	}
	var err error
	cfg := llm.Config{
		Provider:  mc.Provider,
		Model:     mc.Model,
		APIKey:    os.Getenv(mc.APIKeyEnv),
		MaxTokens: maxTokens,
		Timeout:   timeout,
		BaseURL:   mc.BaseURL,
	}
	if len(mc.Capabilities) > 0 {
		if cfg.ModelCapabilities, err = capabilitySet(mc.Capabilities); err != nil {
			return err
		}
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	group := map[string]string{}
	for _, f := range Families {
		group[f.Name] = f.Group
	}

	type job struct {
		t      Task
		sample int
	}
	jobs := make(chan job)
	var done, passed, errs int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < conc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				rec := ask(client, mc, j.t, timeout, maxTokens)
				rec.RunID, rec.Set, rec.Sample = runID, set, j.sample
				rec.Group = group[j.t.Family]
				write(rec)
				mu.Lock()
				done++
				if rec.Correct() {
					passed++
				}
				if rec.Outcome == "error" {
					errs++
				}
				if done%10 == 0 || done == len(tasks)*samples {
					fmt.Fprintf(os.Stderr, "  %s: %d/%d done, %d pass, %d errors\n",
						mc.Model, done, len(tasks)*samples, passed, errs)
				}
				mu.Unlock()
			}
		}()
	}
	for s := 0; s < samples; s++ {
		for _, t := range tasks {
			jobs <- job{t, s}
		}
	}
	close(jobs)
	wg.Wait()
	return nil
}

// ask sends one task (with up to two retries on transport errors) and grades it.
func ask(client llm.Client, mc ModelConfig, t Task, timeout time.Duration, maxTokens int) Record {
	rec := Record{
		Time: time.Now().UTC().Format(time.RFC3339), Provider: mc.Provider, Model: mc.Model,
		Thinking: mc.Thinking, Temperature: mc.Temperature,
		Family: t.Family, Level: t.Level, TaskID: t.ID, Gold: t.Gold,
	}
	msgs := make([]llm.Message, 0, len(t.Turns))
	for _, turn := range t.Turns {
		role := llm.RoleUser
		if turn.Role == "assistant" {
			role = llm.RoleAssistant
		}
		msgs = append(msgs, llm.NewTextMessage(role, turn.Text))
	}
	req := llm.Request{
		Messages:      msgs,
		MaxTokens:     maxTokens,
		Temperature:   mc.Temperature,
		ThinkingLevel: llm.ThinkingLevel(mc.Thinking),
	}

	var resp *llm.Response
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * 5 * time.Second)
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		start := time.Now()
		resp, err = client.Complete(ctx, req)
		rec.LatencyMS = time.Since(start).Milliseconds()
		cancel()
		if err == nil {
			break
		}
	}
	if err != nil {
		rec.Outcome = "error"
		rec.Error = err.Error()
		return rec
	}

	rec.ServedModel = resp.Model
	rec.InputTokens, rec.OutputTokens, rec.ThinkingTokens = resp.InputTokens, resp.OutputTokens, resp.ThinkingTokens
	rec.ThinkingChars = len(resp.Thinking)
	rec.StopReason = resp.StopReason
	rec.Response = truncate(resp.Content, 4000)

	correct, extracted, found := t.Check(resp.Content)
	rec.Extracted = truncate(extracted, 300)
	switch {
	case correct:
		rec.Outcome = "pass"
	case resp.StopReason == "refusal":
		rec.Outcome = "refusal"
	case resp.StopReason == "max_tokens":
		rec.Outcome = "truncated"
	case !found:
		rec.Outcome = "no_answer"
	default:
		rec.Outcome = "fail"
	}
	return rec
}

// =============================================================================
// summary / compare
// =============================================================================

func loadRecords(paths []string) ([]Record, error) {
	var out []Record
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 16<<20)
		line := 0
		for sc.Scan() {
			line++
			if strings.TrimSpace(sc.Text()) == "" {
				continue
			}
			var r Record
			if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
				f.Close()
				return nil, fmt.Errorf("%s:%d: %w", p, line, err)
			}
			out = append(out, r)
		}
		err = sc.Err()
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
	}
	return out, nil
}

// stat aggregates records of one (model, family) cell. Errors and refusals are
// counted but excluded from accuracy: an outage or a safety refusal is not a
// reasoning failure (both are reported, and a refusal-rate change is a signal).
type stat struct {
	pass, graded, errors, refusals, truncated, noAnswer int
	outTokens, latency                                  []float64
	served                                              map[string]int
	byTask                                              map[string]bool
}

func newStat() *stat { return &stat{served: map[string]int{}, byTask: map[string]bool{}} }

func (s *stat) add(r Record) {
	if r.Outcome == "error" {
		s.errors++
		return
	}
	if r.Outcome == "refusal" {
		s.refusals++
		return
	}
	s.graded++
	if r.Correct() {
		s.pass++
	}
	switch r.Outcome {
	case "truncated":
		s.truncated++
	case "no_answer":
		s.noAnswer++
	}
	s.outTokens = append(s.outTokens, float64(r.OutputTokens+r.ThinkingTokens))
	s.latency = append(s.latency, float64(r.LatencyMS))
	s.served[r.ServedModel]++
	s.byTask[fmt.Sprintf("%s#%d", r.TaskID, r.Sample)] = r.Correct()
}

func (s *stat) acc() float64 {
	if s.graded == 0 {
		return 0
	}
	return float64(s.pass) / float64(s.graded)
}

type cellKey struct{ model, family string }

// aggregate groups records per model × family, plus per model × group and
// per model overall ("*all*").
func aggregate(recs []Record) map[cellKey]*stat {
	m := map[cellKey]*stat{}
	get := func(k cellKey) *stat {
		if m[k] == nil {
			m[k] = newStat()
		}
		return m[k]
	}
	for _, r := range recs {
		model := r.Provider + "/" + r.Model
		get(cellKey{model, r.Family}).add(r)
		get(cellKey{model, "[" + r.Group + "]"}).add(r)
		get(cellKey{model, "*all*"}).add(r)
	}
	return m
}

func sortedKeys(m map[cellKey]*stat) []cellKey {
	order := map[string]int{"*all*": 0, "[intent]": 1, "[agentic]": 2, "[precision]": 3}
	for i, f := range Families {
		order[f.Name] = 10 + i
	}
	keys := make([]cellKey, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].model != keys[j].model {
			return keys[i].model < keys[j].model
		}
		return order[keys[i].family] < order[keys[j].family]
	})
	return keys
}

func cmdSummary(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("summary needs at least one JSONL file")
	}
	return summarize(os.Stdout, args)
}

func summarize(out io.Writer, paths []string) error {
	recs, err := loadRecords(paths)
	if err != nil {
		return err
	}
	cells := aggregate(recs)
	fmt.Fprintln(out, "| model | family | n | accuracy | trunc | no-answer | refusals | errors | median out+think tokens | median latency | served model |")
	fmt.Fprintln(out, "|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|")
	for _, k := range sortedKeys(cells) {
		s := cells[k]
		fmt.Fprintf(out, "| %s | %s | %d | %.0f%% | %d | %d | %d | %d | %.0f | %.1fs | %s |\n",
			k.model, k.family, s.graded, 100*s.acc(), s.truncated, s.noAnswer, s.refusals, s.errors,
			median(s.outTokens), median(s.latency)/1000, servedList(s.served))
	}
	return nil
}

func servedList(m map[string]int) string {
	var xs []string
	for k, v := range m {
		if k == "" {
			k = "?"
		}
		xs = append(xs, fmt.Sprintf("%s×%d", k, v))
	}
	sort.Strings(xs)
	return strings.Join(xs, " ")
}

func cmdCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	base := fs.String("base", "", "baseline JSONL file(s), comma-separated")
	cand := fs.String("cand", "", "candidate JSONL file(s), comma-separated")
	alpha := fs.Float64("alpha", 0.01, "significance level for a DRIFT verdict")
	fs.Parse(args)
	if *base == "" || *cand == "" {
		return fmt.Errorf("compare needs -base and -cand")
	}
	b, err := loadRecords(splitList(*base))
	if err != nil {
		return err
	}
	c, err := loadRecords(splitList(*cand))
	if err != nil {
		return err
	}
	drift := compare(os.Stdout, aggregate(b), aggregate(c), *alpha)
	if drift {
		os.Exit(3) // usable as a cron/CI alarm
	}
	return nil
}

// compare prints one row per cell present in both runs and reports whether
// any cell shows a significant accuracy drop.
func compare(out io.Writer, base, cand map[cellKey]*stat, alpha float64) bool {
	fmt.Fprintln(out, "| model | family | base acc (n) | cand acc (n) | Δ | test | p | tokens base→cand | p tokens | latency base→cand | refusals base→cand | served model | verdict |")
	fmt.Fprintln(out, "|---|---|---|---|---:|---|---:|---|---:|---|---|---|---|")
	anyDrift := false
	for _, k := range sortedKeys(cand) {
		bs, cs := base[k], cand[k]
		if bs == nil || bs.graded+bs.refusals == 0 || cs.graded+cs.refusals == 0 {
			continue
		}
		// Paired comparison when both runs answered the same tasks.
		var b10, c01, pairs int
		for id, ok := range bs.byTask {
			if ok2, both := cs.byTask[id]; both {
				pairs++
				if ok && !ok2 {
					b10++
				}
				if !ok && ok2 {
					c01++
				}
			}
		}
		test, p := "fisher", FisherExact(bs.pass, bs.graded-bs.pass, cs.pass, cs.graded-cs.pass)
		if pairs >= 10 {
			test, p = fmt.Sprintf("mcnemar %d↓/%d↑", b10, c01), McNemarExact(b10, c01)
		}
		delta := cs.acc() - bs.acc()
		pTok := MannWhitney(bs.outTokens, cs.outTokens)
		verdict := ""
		switch {
		case p < alpha && delta < 0:
			verdict = "**DRIFT ↓**"
			anyDrift = true
		case p < 0.05 && delta < 0:
			verdict = "warn ↓"
		case p < alpha && delta > 0:
			verdict = "better ↑"
		}
		mb, mc := median(bs.outTokens), median(cs.outTokens)
		if pTok < alpha && mb > 0 && mc < 0.8*mb {
			verdict = strings.TrimSpace(verdict + " fewer tokens")
		}
		nb, nc := bs.graded+bs.refusals, cs.graded+cs.refusals
		if FisherExact(bs.refusals, nb-bs.refusals, cs.refusals, nc-cs.refusals) < alpha && cs.refusals*nb > bs.refusals*nc {
			verdict = strings.TrimSpace(verdict + " more refusals")
		}
		servedB, servedC := servedList(bs.served), servedList(cs.served)
		served := servedC
		if names(bs.served) != names(cs.served) {
			served = "**changed:** " + servedB + " → " + servedC
		}
		fmt.Fprintf(out, "| %s | %s | %.0f%% (%d) | %.0f%% (%d) | %+.0f pp | %s | %.3g | %.0f→%.0f | %.3g | %.1fs→%.1fs | %d→%d | %s | %s |\n",
			k.model, k.family, 100*bs.acc(), bs.graded, 100*cs.acc(), cs.graded, 100*delta, test, p,
			mb, mc, pTok, median(bs.latency)/1000, median(cs.latency)/1000, bs.refusals, cs.refusals, served, verdict)
	}
	return anyDrift
}

func names(m map[string]int) string {
	var xs []string
	for k := range m {
		xs = append(xs, k)
	}
	sort.Strings(xs)
	return strings.Join(xs, ",")
}

// =============================================================================
// show
// =============================================================================

func cmdShow(args []string) error {
	fs := flag.NewFlagSet("show", flag.ExitOnError)
	family := fs.String("family", "intent", "task family")
	level := fs.Int("level", 3, "difficulty 1..5")
	idx := fs.Int("idx", 0, "task index")
	set := fs.String("set", "fixed", "seed set")
	fs.Parse(args)
	s := *set
	if s == "fixed" {
		s = FixedSet
	}
	t, err := Generate(s, *family, *level, *idx)
	if err != nil {
		return err
	}
	fmt.Printf("# %s\n\n", t.ID)
	for _, turn := range t.Turns {
		fmt.Printf("--- %s ---\n%s\n\n", strings.ToUpper(turn.Role), turn.Text)
	}
	fmt.Printf("--- GOLD ---\n%s\n", t.Gold)
	return nil
}

// =============================================================================
// small helpers
// =============================================================================

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func matchAny(s string, subs []string) bool {
	if len(subs) == 0 {
		return true
	}
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
