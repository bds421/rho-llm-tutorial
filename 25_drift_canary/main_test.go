package main

import (
	"fmt"
	"io"
	"math"
	"strings"
	"testing"
)

// goldResponse builds a model reply that a perfect model would give.
func goldResponse(t Task) string {
	if t.Family == "instruct_persist" {
		return "GOAL: Count how often the hourly job runs.\nThe job runs once per hour including both ends.\nANSWER: " + t.Gold
	}
	return "Working through it step by step.\n\nANSWER: " + t.Gold
}

// Every family, every level, many seeds: the gold answer must pass its own
// checker. A checker that rejects the truth would look exactly like a nerf.
func TestGoldAnswersPass(t *testing.T) {
	for _, f := range Families {
		for level := 1; level <= 5; level++ {
			for idx := 0; idx < 40; idx++ {
				task, err := Generate("test", f.Name, level, idx)
				if err != nil {
					t.Fatal(err)
				}
				ok, got, found := task.Check(goldResponse(task))
				if !ok || !found {
					t.Fatalf("%s: gold %q rejected (extracted %q)\n%s", task.ID, task.Gold, got, task.Turns[len(task.Turns)-1].Text)
				}
			}
		}
	}
}

func TestGenerationIsDeterministic(t *testing.T) {
	for _, f := range Families {
		a, _ := Generate("s", f.Name, 3, 7)
		b, _ := Generate("s", f.Name, 3, 7)
		c, _ := Generate("other", f.Name, 3, 7)
		if fmt.Sprint(a.Turns) != fmt.Sprint(b.Turns) || a.Gold != b.Gold {
			t.Errorf("%s: same seed produced different tasks", f.Name)
		}
		if fmt.Sprint(a.Turns) == fmt.Sprint(c.Turns) {
			t.Errorf("%s: different seed sets produced the same task", f.Name)
		}
	}
}

func TestGenerateRejectsBadInput(t *testing.T) {
	if _, err := Generate("s", "nope", 3, 0); err == nil {
		t.Error("unknown family accepted")
	}
	for _, l := range []int{0, 6, -1} {
		if _, err := Generate("s", "intent", l, 0); err == nil {
			t.Errorf("level %d accepted", l)
		}
	}
}

func TestNoAnswerLineIsNotFound(t *testing.T) {
	for _, f := range Families {
		task, _ := Generate("s", f.Name, 3, 0)
		ok, _, found := task.Check("I think the answer is " + task.Gold)
		if ok || found {
			t.Errorf("%s: reply without ANSWER line graded ok=%v found=%v", f.Name, ok, found)
		}
	}
}

func TestLastAnswerLineWins(t *testing.T) {
	task, _ := Generate("s", "arith_mul", 2, 0)
	resp := "ANSWER: " + task.Gold + "\nWait, let me recheck.\nANSWER: 12345"
	if ok, _, _ := task.Check(resp); ok {
		t.Error("an earlier correct ANSWER line rescued a wrong final answer")
	}
	resp = "ANSWER: 12345\nWait, I made an error.\n**ANSWER:** `" + task.Gold + "`"
	if ok, _, _ := task.Check(resp); !ok {
		t.Error("markdown-decorated final answer rejected")
	}
}

// Wrong but plausible answers must fail: substring matching is not enough.
func TestNearMissAnswersFail(t *testing.T) {
	for idx := 0; idx < 20; idx++ {
		task, _ := Generate("s", "arith_mul", 3, idx)
		wrong := task.Gold[:len(task.Gold)-1] + string(rune('0'+(task.Gold[len(task.Gold)-1]-'0'+1)%10))
		if ok, _, _ := task.Check("ANSWER: " + wrong); ok {
			t.Fatalf("off-by-one-digit product accepted: %s vs %s", wrong, task.Gold)
		}
		if ok, _, _ := task.Check("ANSWER: 1" + task.Gold); ok {
			t.Fatal("gold as substring of a longer number accepted")
		}
		rev, _ := Generate("s", "str_reverse", 3, idx)
		if ok, _, _ := rev.Check("ANSWER: " + strings.ToLower(rev.Gold)); ok && rev.Gold != strings.ToLower(rev.Gold) {
			t.Fatal("case-folded reversal accepted")
		}
	}
}

// The trap from the real report: the account column is not the password owner.
func TestLogCountRunAccountTrapFails(t *testing.T) {
	for idx := 0; idx < 20; idx++ {
		task, _ := Generate("s", "log_count", 3, idx)
		prompt := task.Turns[0].Text
		var acct string
		for _, a := range runAccounts {
			if strings.Contains(task.Gold, a) {
				t.Fatalf("gold names a run account: %s", task.Gold)
			}
			if strings.Contains(prompt, ","+a+",importer.exe,importer.exe /db:sales") {
				acct = a
			}
		}
		if acct == "" {
			t.Fatal("could not find the run account in the records")
		}
		trap := strings.Replace(task.Gold, `"password_login":"`, `"password_login":"`+acct+`","x":"`, 1)
		if ok, _, _ := task.Check("ANSWER: " + trap); ok {
			t.Fatalf("run account accepted as password owner: %s", trap)
		}
	}
}

func TestCLIFlagsGoSemantics(t *testing.T) {
	var task Task
	for idx := 0; ; idx++ {
		task, _ = Generate("s", "cli_flags", 5, idx)
		if strings.Contains(task.Gold, "-verify=false") {
			break
		}
	}
	if ok, _, _ := task.Check("ANSWER: " + task.Gold); !ok {
		t.Fatal("gold command rejected")
	}
	if ok, _, _ := task.Check("ANSWER: " + strings.Replace(task.Gold, "-verify=false", "-verify false", 1)); ok {
		t.Error(`"-verify false" accepted; Go flags treat it as verify=true plus a positional argument`)
	}
	if ok, _, _ := task.Check("ANSWER: " + strings.Replace(task.Gold, "-passes 1", "", 1)); ok {
		t.Error("command relying on the default passes accepted")
	}
	if ok, _, _ := task.Check("ANSWER: " + task.Gold + " -dry-run"); ok {
		t.Error("dry-run command accepted")
	}
	if ok, _, _ := task.Check("ANSWER: " + strings.Replace(task.Gold, "-passes 1", "--passes=1", 1)); !ok {
		t.Error("equivalent --passes=1 rejected")
	}
}

func TestProgressStateNoneVsList(t *testing.T) {
	sawNone, sawList := false, false
	for idx := 0; idx < 60; idx++ {
		task, _ := Generate("s", "progress_state", 3, idx)
		if task.Gold == "none" {
			sawNone = true
			if ok, _, _ := task.Check("ANSWER: day-01"); ok {
				t.Fatal("non-empty list accepted when everything is finished")
			}
		} else {
			sawList = true
			if ok, _, _ := task.Check("ANSWER: none"); ok {
				t.Fatal("'none' accepted when items are unfinished")
			}
			first := strings.Split(task.Gold, ",")[0]
			if strings.Contains(task.Gold, ",") {
				if ok, _, _ := task.Check("ANSWER: " + first); ok {
					t.Fatal("partial list accepted")
				}
			}
		}
	}
	if !sawNone || !sawList {
		t.Errorf("expected both all-done and unfinished cases (none=%v list=%v)", sawNone, sawList)
	}
}

func TestInstructPersistRules(t *testing.T) {
	task, _ := Generate("s", "instruct_persist", 3, 0)
	cases := map[string]string{
		"no GOAL line":   "The job runs often.\nANSWER: " + task.Gold,
		"German reply":   "GOAL: Zählen.\nDer Job läuft jeden Tag und das ist nicht wenig, mit Pausen.\nANSWER: " + task.Gold,
		"bullet list":    "GOAL: Count runs.\n- one run per hour\nANSWER: " + task.Gold,
		"numbered list":  "GOAL: Count runs.\n1. hours inclusive\nANSWER: " + task.Gold,
		"table":          "GOAL: Count runs.\n| h | n |\nANSWER: " + task.Gold,
		"fencepost trap": "GOAL: Count runs.\nIt runs every hour.\nANSWER: 1",
	}
	for name, resp := range cases {
		if ok, _, _ := task.Check(resp); ok {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestIntentLetters(t *testing.T) {
	counts := map[string]int{}
	for idx := 0; idx < 200; idx++ {
		task, _ := Generate("s", "intent", 4, idx)
		counts[task.Gold]++
		wrong := "A"
		if task.Gold == "A" {
			wrong = "B"
		}
		if ok, _, _ := task.Check("ANSWER: " + wrong); ok {
			t.Fatal("wrong letter accepted")
		}
		for _, form := range []string{"**" + task.Gold + "**", task.Gold + ")", strings.ToLower(task.Gold), "Option " + task.Gold} {
			if ok, _, _ := task.Check("ANSWER: " + form); !ok {
				t.Fatalf("answer form %q rejected", form)
			}
		}
	}
	for _, l := range []string{"A", "B", "C", "D"} {
		if counts[l] < 25 {
			t.Errorf("correct letter %s appears only %d/200 times; position bias would leak", l, counts[l])
		}
	}
}

func TestTypoKeepsUTF8(t *testing.T) {
	r := seededRand("utf8")
	for i := 0; i < 200; i++ {
		out := typo(r, "nein, der quick check ist für den nächsten schritt größer", 1.0)
		if !strings.ContainsRune(out, 'f') || strings.ContainsRune(out, '�') {
			t.Fatalf("typo broke the text: %q", out)
		}
		for _, c := range out {
			if c == '�' {
				t.Fatalf("invalid UTF-8 after typo: %q", out)
			}
		}
	}
}

func TestParseFlags(t *testing.T) {
	if parseFlags("rm -rf /") != nil {
		t.Error("other command accepted")
	}
	if parseFlags("derive -in") != nil {
		t.Error("missing value accepted")
	}
	f := parseFlags(`derive -in "a.parquet" --workers=4 -verify`)
	if f["in"] != "a.parquet" || f["workers"] != "4" || f["verify"] != "true" {
		t.Errorf("unexpected parse: %v", f)
	}
}

// =============================================================================
// statistics
// =============================================================================

func near(a, b float64) bool { return math.Abs(a-b) < 1e-3 }

func TestFisherExact(t *testing.T) {
	// Classic tea-tasting table.
	if p := FisherExact(3, 1, 1, 3); !near(p, 0.4857) {
		t.Errorf("tea tasting p = %v, want 0.4857", p)
	}
	if p := FisherExact(10, 0, 0, 10); p > 1e-4 {
		t.Errorf("perfect separation p = %v", p)
	}
	if p := FisherExact(0, 0, 0, 0); p != 1 {
		t.Errorf("empty table p = %v", p)
	}
	if p := FisherExact(50, 50, 50, 50); !near(p, 1) {
		t.Errorf("identical rows p = %v", p)
	}
}

func TestMcNemarExact(t *testing.T) {
	if p := McNemarExact(0, 10); !near(p, 2.0/1024) {
		t.Errorf("p = %v, want %v", p, 2.0/1024)
	}
	if p := McNemarExact(5, 5); p != 1 {
		t.Errorf("balanced p = %v", p)
	}
	if p := McNemarExact(0, 0); p != 1 {
		t.Errorf("no discordant pairs p = %v", p)
	}
}

func TestMannWhitney(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if p := MannWhitney(x, x); p < 0.9 {
		t.Errorf("identical samples p = %v", p)
	}
	y := []float64{101, 102, 103, 104, 105, 106, 107, 108, 109, 110}
	if p := MannWhitney(x, y); p > 0.001 {
		t.Errorf("separated samples p = %v", p)
	}
	if p := MannWhitney([]float64{5, 5, 5}, []float64{5, 5, 5}); p != 1 {
		t.Errorf("all ties p = %v", p)
	}
	if p := MannWhitney(nil, x); p != 1 {
		t.Errorf("empty sample p = %v", p)
	}
}

// A simulated 20-point drop on the fixed set must be flagged; identical runs must not.
func TestCompareFlagsDrop(t *testing.T) {
	mk := func(passEvery int) []Record {
		var recs []Record
		for i := 0; i < 100; i++ {
			out := "fail"
			if i%passEvery != 0 {
				out = "pass"
			}
			recs = append(recs, Record{Provider: "p", Model: "m", Family: "intent", Group: "intent",
				TaskID: fmt.Sprintf("intent/L3/fixed/%03d", i), Outcome: out, OutputTokens: 100})
		}
		return recs
	}
	base := aggregate(mk(10)) // 90%
	cand := aggregate(mk(3))  // ~67%
	devnull := io.Discard
	if !compare(devnull, base, cand, 0.01) {
		t.Error("a 23-point paired drop was not flagged")
	}
	if compare(devnull, base, aggregate(mk(10)), 0.01) {
		t.Error("identical runs flagged as drift")
	}
}

func TestErrorsAndRefusalsExcludedFromAccuracy(t *testing.T) {
	s := newStat()
	s.add(Record{Outcome: "pass"})
	s.add(Record{Outcome: "error"})
	s.add(Record{Outcome: "error"})
	s.add(Record{Outcome: "refusal"})
	if s.acc() != 1 || s.errors != 2 || s.refusals != 1 || s.graded != 1 {
		t.Errorf("acc=%v errors=%d refusals=%d graded=%d", s.acc(), s.errors, s.refusals, s.graded)
	}
}

func TestCompareFlagsRefusalSpike(t *testing.T) {
	mk := func(refuseEvery int) []Record {
		var recs []Record
		for i := 0; i < 100; i++ {
			out := "pass"
			if i%refuseEvery == 0 {
				out = "refusal"
			}
			recs = append(recs, Record{Provider: "p", Model: "m", Family: "log_count", Group: "agentic",
				TaskID: fmt.Sprintf("log_count/L3/fixed/%03d", i), Outcome: out})
		}
		return recs
	}
	var sb strings.Builder
	compare(&sb, aggregate(mk(50)), aggregate(mk(3)), 0.01)
	if !strings.Contains(sb.String(), "more refusals") {
		t.Errorf("refusal spike 2%%→34%% not flagged:\n%s", sb.String())
	}
}
