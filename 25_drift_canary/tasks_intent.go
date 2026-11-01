package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// =============================================================================
// Intent family — does the model understand what the user actually means?
//
// Every scenario is distilled from a real agent session where the user had to
// correct the model. The user turn is terse and informal (typos, German/English
// mix, references to earlier turns), exactly like real chat. The distractors are
// the misreadings the model actually produced in those sessions:
//   - doing the deferred step now ("the quick check is for the NEXT step")
//   - imposing its own style when told to keep the current one
//   - defending stale numbers instead of treating them as an old baseline
//   - asking a confirmation question the user already answered
//   - finishing an expensive run when told to stop and do a POC
//   - acting on the wrong referent ("the second text")
//   - implementing instead of recording a design question as a TODO
//   - reusing the old data source when a new one exists
// =============================================================================

type intentScenario struct {
	context   func(r *rand.Rand) string // previous assistant turn
	userEN    string
	userDE    string // mixed-language variant (used at level >= 3)
	correct   string
	wrong     []string
	extraRule string // optional extra instruction from earlier in the session
}

var projects = []string{"the indexer", "the reasoner", "the report pipeline", "the ingest job", "the cross-check"}
var datasets = []string{"the reduced data set", "the twelve-day set", "the masked sample", "the March export"}

var intentScenarios = []intentScenario{
	{ // deferred step
		context: func(r *rand.Rand) string {
			return fmt.Sprintf("Plan:\n1. Run the quick check on %s.\n2. Then analyse what the next best step is.\nStarting with step 1 now.", pick(r, datasets))
		},
		userEN:  "no, the quick check is for the next step.. for now lets analyse what the next best step is. do it interactively.. first state the goal back to me",
		userDE:  "nein, der quick check ist für den nächsten schritt.. for now lets analyse what the next best step is. do it interactively.. first state the goal back to me",
		correct: "Do not run the quick check; restate the goal in my own words and wait for confirmation before analysing the next step",
		wrong: []string{
			"Run the quick check now, then analyse the next step",
			"Run the quick check first because it is fast, then state the goal",
			"Skip the goal statement and present a full analysis of the next step right away",
		},
	},
	{ // keep current style
		context: func(r *rand.Rand) string {
			return "I compared the two versions of the report. Mine reads better and I would restore it. The current version has 15 wrong items and 14 unsupported claims."
		},
		userEN:  "NO, yours was NOT better to read (it was actually horrible to read). fix all proven wrong items.. but lets take the writing style as we have right now, not your own",
		userDE:  "NEIN, deine version war NICHT besser zu lesen. fix all proven wrong items.. aber nimm den schreibstil den wir jetzt haben, nicht deinen eigenen",
		correct: "Fix only the proven wrong items inside the current version and keep its wording and style",
		wrong: []string{
			"Restore my version and fix the wrong items there",
			"Rewrite the whole report in a clearer style and fix the wrong items",
			"Fix the wrong items and also the unsupported claims, rewriting paragraphs where needed",
		},
	},
	{ // stale numbers
		context: func(r *rand.Rand) string {
			return fmt.Sprintf("The plan is not reachable: measured read throughput is %d MB/s, so the full corpus needs %d hours, far beyond the 12-hour target.", 3+r.IntN(5), 70+r.IntN(30))
		},
		userEN:  "no, you are wrong, these are wrong numbers.. from before.. we want to increase the read numbers in the next project phase",
		userDE:  "nein, das sind falsche zahlen.. von vorher.. we want to increase the read numbers in the next project phase",
		correct: "Treat those numbers as the old baseline and plan the work that raises read throughput in the next phase",
		wrong: []string{
			"Re-measure with the same setup to prove the numbers are correct",
			"Explain again why the numbers make the plan unreachable",
			"Drop the 12-hour target as infeasible",
		},
	},
	{ // needless confirmation question
		context: func(r *rand.Rand) string {
			return fmt.Sprintf("All tests pass for %s. You said earlier to commit once the tests are green. Shall I commit?", pick(r, projects))
		},
		userEN:  "YES commit. this is a stupid question.. then lets get to work",
		userDE:  "JA commit. das ist eine dumme frage.. dann lets get to work",
		correct: "Commit and continue directly with the next open task without asking again",
		wrong: []string{
			"Commit and ask which task to work on next",
			"Ask whether the commit should also be pushed",
			"Explain why the confirmation question was necessary",
		},
	},
	{ // stop and POC
		context: func(r *rand.Rand) string {
			return fmt.Sprintf("The full pass over %s started; it takes about %d minutes. I'll report when it finishes.", pick(r, datasets), 80+r.IntN(60))
		},
		userEN:  "stop tis.. start again with a poc",
		userDE:  "stopp das.. start again with a poc",
		correct: "Stop the running full pass and build a small proof of concept on a tiny sample",
		wrong: []string{
			"Let the full pass finish, then build a proof of concept",
			"Restart the full pass with fixes",
			"Write a design document for a proof of concept",
		},
	},
	{ // referent resolution
		context: func(r *rand.Rand) string {
			return "Here are two drafts of the summary.\n\nText 1: (a 120-word version focusing on the timeline)\n\nText 2: (a 260-word version with all measured numbers)"
		},
		userEN:  "the second text is better but way too long.. make it half as long, keep the numbers",
		userDE:  "der zweite text ist besser aber viel zu lang.. make it half as long, keep the numbers",
		correct: "Shorten Text 2 to about half its length and keep its numbers",
		wrong: []string{
			"Shorten Text 1 to about half its length",
			"Merge both texts into one shorter text",
			"Shorten Text 2 by removing the numbers",
		},
	},
	{ // design question -> todo, not implementation
		context: func(r *rand.Rand) string {
			return fmt.Sprintf("Implementing the memory fix for %s now: step 3 of 5, adding a cache layer.", pick(r, projects))
		},
		userEN:  "should the reasoner not have just the data for the current reasoning in memory? and the rest on fast disk? write this item in the todo.md as the first step with tests first. then analyse the question with a separate agent",
		userDE:  "sollte der reasoner nicht nur die daten für das aktuelle reasoning im memory haben? rest auf fast disk? write this item in the todo.md as the first step with tests first. then analyse the question with a separate agent",
		correct: "Stop the cache work, add the item as the first todo with tests first, then start a separate agent to analyse the design question",
		wrong: []string{
			"Finish the cache layer first, then add the todo item",
			"Implement the keep-only-current-data change right away",
			"Answer the design question in chat and continue with the cache layer",
		},
	},
	{ // new data source
		context: func(r *rand.Rand) string {
			return "Deep analysis finished on the gen3 index: three hosts stand out, details in deep-reading.md."
		},
		userEN:  "respawn the deep analysis agent again.. this was not a good analysis.. why the old index? we have a new index?",
		userDE:  "starte den deep analysis agent nochmal.. das war keine gute analyse.. warum der alte index? wir haben doch einen neuen index?",
		correct: "Rerun the deep analysis on the new index",
		wrong: []string{
			"Rerun the deep analysis on the gen3 index with more detail",
			"Explain why the gen3 index was the right choice",
			"Merge the gen3 results with a quick look at the new index",
		},
	},
}

// typo injects human typing noise: dropped, doubled and swapped letters.
func typo(r *rand.Rand, s string, rate float64) string {
	words := strings.Fields(s)
	for i, w := range words {
		b := []rune(w)
		if len(b) < 4 || r.Float64() >= rate {
			continue
		}
		p := 1 + r.IntN(len(b)-2)
		switch r.IntN(3) {
		case 0:
			b = append(b[:p], b[p+1:]...)
		case 1:
			b = append(b[:p+1], b[p:]...)
		default:
			b[p], b[p+1] = b[p+1], b[p]
		}
		words[i] = string(b)
	}
	return strings.Join(words, " ")
}

// noise returns tool-output-like lines that bury the context (level >= 4).
func noise(r *rand.Rand, n int) string {
	var sb strings.Builder
	sb.WriteString("[tool output]\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, "2026-09-%02d %02d:%02d:%02d worker-%d status=%s rows=%d rss=%dMiB\n",
			1+r.IntN(28), r.IntN(24), r.IntN(60), r.IntN(60), r.IntN(8),
			pick(r, []string{"ok", "ok", "ok", "retry", "slow"}), r.IntN(5_000_000), 500+r.IntN(90_000))
	}
	sb.WriteString("[end tool output]\n\n")
	return sb.String()
}

func genIntent(r *rand.Rand, level int) Task {
	sc := intentScenarios[r.IntN(len(intentScenarios))]
	ctx := sc.context(r)
	if level >= 4 {
		ctx = noise(r, 10*(level-3)) + ctx
	}
	msg := sc.userEN
	if level >= 3 {
		msg = sc.userDE
	}
	if level >= 2 {
		msg = typo(r, msg, 0.1*float64(level-1))
	}
	options := append([]string{sc.correct}, sc.wrong...)
	optText, gold := letterOptions(r, options, 0)

	question := fmt.Sprintf(
		"You are a coding agent working with this user. Read the user's last message carefully and decide what the user wants you to do next.\n\n"+
			"Possible next actions:\n%s\nDo not explain at length. %s The answer is the letter of the action.",
		optText, answerRule)

	turns := []Turn{
		{Role: "user", Text: "Continue with the current task."},
		{Role: "assistant", Text: ctx},
		{Role: "user", Text: msg + "\n\n---\n" + question},
	}
	return Task{Turns: turns, Gold: gold, check: checkLetter(gold)}
}

// =============================================================================
// Instruction persistence — rules from turn 1 still apply many turns later,
// even when the user switches language ("i asked already more than twice").
// =============================================================================

var germanWords = map[string]bool{"und": true, "der": true, "die": true, "das": true, "ist": true, "nicht": true,
	"ein": true, "eine": true, "mit": true, "für": true, "von": true, "zu": true, "den": true, "dem": true,
	"wird": true, "sind": true, "auch": true, "sich": true, "läuft": true, "pro": true, "mal": true, "jeden": true}

func genInstructPersist(r *rand.Rand, level int) Task {
	rules := "For the rest of this conversation, follow these rules in every reply:\n" +
		"1. Always answer in English, even when I write in another language.\n" +
		"2. The first line of every reply starts with `GOAL:` followed by one sentence restating what I asked.\n" +
		"3. Never use bullet lists, numbered lists or tables. Plain sentences only.\n" +
		"4. " + answerRule
	turns := []Turn{
		{Role: "user", Text: rules},
		{Role: "assistant", Text: "GOAL: Confirm the four rules for this conversation.\nUnderstood, I will follow all four rules in every reply.\nANSWER: ok"},
	}
	fillers := []struct{ q, a string }{
		{"How many minutes are in 2.5 hours?", "GOAL: Convert 2.5 hours to minutes.\nOne hour has 60 minutes, so 2.5 hours have 150 minutes.\nANSWER: 150"},
		{"What is 15% of 240?", "GOAL: Compute 15 percent of 240.\nTen percent is 24 and five percent is 12, which adds up to 36.\nANSWER: 36"},
		{"How many days are in three weeks?", "GOAL: Count the days in three weeks.\nA week has seven days, so three weeks have 21 days.\nANSWER: 21"},
		{"If a file is 3 GB and I copy it at 100 MB/s, how many seconds does it take?", "GOAL: Compute the copy time for 3 GB at 100 MB/s.\nThree gigabytes are 3000 megabytes, which take 30 seconds at 100 MB/s.\nANSWER: 30"},
	}
	for i := 0; i < level && i < len(fillers); i++ {
		f := fillers[(i+r.IntN(len(fillers)))%len(fillers)]
		turns = append(turns, Turn{"user", f.q}, Turn{"assistant", f.a})
	}

	// Fencepost question: hourly runs from H1:MM to H2:MM inclusive.
	h1 := 5 + r.IntN(4)
	h2 := 15 + r.IntN(5)
	mm := 5 + r.IntN(50)
	runsPerDay := h2 - h1 + 1
	var q string
	gold := runsPerDay
	if level >= 3 {
		days := 2 + r.IntN(10)
		gold = runsPerDay * days
		q = fmt.Sprintf("Ein Job läuft jede Stunde, der erste Lauf um %02d:%02d, der letzte um %02d:%02d. Wie oft läuft er insgesamt in %d Tagen?", h1, mm, h2, mm, days)
	} else {
		q = fmt.Sprintf("Ein Job läuft jede Stunde, der erste Lauf um %02d:%02d, der letzte um %02d:%02d. Wie oft läuft er pro Tag?", h1, mm, h2, mm)
	}
	turns = append(turns, Turn{"user", q})
	g := fmt.Sprint(gold)

	check := func(resp string) (bool, string, bool) {
		got, found := answerLine(resp)
		if !found {
			return false, "", false
		}
		ok := digitsOnly(got) == g
		lines := strings.Split(strings.TrimSpace(resp), "\n")
		if !strings.HasPrefix(strings.TrimLeft(lines[0], "*# "), "GOAL:") {
			ok = false
			got += " [no GOAL line]"
		}
		for _, l := range lines {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, "|") || strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") ||
				(len(t) > 2 && t[0] >= '1' && t[0] <= '9' && (t[1] == '.' || t[1] == ')')) {
				ok = false
				got += " [list/table]"
				break
			}
		}
		de := 0
		for _, w := range strings.Fields(strings.ToLower(resp)) {
			if germanWords[strings.Trim(w, ".,:;!?()")] {
				de++
			}
		}
		if de >= 3 {
			ok = false
			got += " [not English]"
		}
		return ok, got, true
	}
	return Task{Turns: turns, Gold: g, check: check}
}
