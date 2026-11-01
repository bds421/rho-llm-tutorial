package main

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
)

// =============================================================================
// Agentic families — the everyday work of a coding/analysis agent: reading
// records faithfully, tracking what is current, noticing what is done, using a
// tool's flags correctly, and choosing the cheap path. All data is synthetic.
// =============================================================================

var hosts = []string{"ws-alpha-01", "ws-bravo-02", "ws-delta-03", "srv-echo-04", "srv-fox-05", "app-golf-06", "app-hotel-07"}
var runAccounts = []string{"svc_report", "svc_batch", "adm_ops", "svc_sync", "local_admin"}
var logins = []string{"imp_batch", "etl_reader", "db_loader", "rpt_writer", "sync_agent"}
var procs = []string{"backup.exe", "sync.exe", "updater.exe", "scan.exe", "rotate.exe", "collect.exe"}

// genLogCount: count runs, time window and credential owner from raw records
// (the report said "44 runs, hourly" when there were 156 runs 05:30–17:30, and
// named the wrong account for the password in the command line).
func genLogCount(r *rand.Rand, level int) Task {
	type row struct{ ts, host, acct, proc, cmd string }
	var rows []row
	days := 2 + level
	h1, h2 := 5+r.IntN(3), 15+r.IntN(4)
	mm := 1 + r.IntN(58)
	host := pick(r, hosts)
	acct := pick(r, runAccounts)
	login := pick(r, logins)
	pw := fmt.Sprintf("Pw%04d!x", r.IntN(10000))
	cmd := fmt.Sprintf("importer.exe /db:sales /login:%s /pwd:%s", login, pw)

	runs := 0
	first := ""
	minTOD, maxTOD := "99:99", "00:00"
	for d := 1; d <= days; d++ {
		for h := h1; h <= h2; h++ {
			// From level 2 on, some runs are missing, so the count must be read
			// from the records instead of computed from the schedule.
			if level >= 2 && r.IntN(9) == 0 {
				continue
			}
			ts := fmt.Sprintf("2026-03-%02d %02d:%02d:%02d", d, h, mm, r.IntN(3))
			rows = append(rows, row{ts, host, acct, "importer.exe", cmd})
			runs++
			if first == "" || ts < first {
				first = ts
			}
			tod := ts[11:16]
			if tod < minTOD {
				minTOD = tod
			}
			if tod > maxTOD {
				maxTOD = tod
			}
		}
	}
	// Noise: other processes, plus decoys from level 3 (same process on another
	// host, and a different process on the same host).
	nNoise := 20 * level
	for i := 0; i < nNoise; i++ {
		h := pick(r, hosts)
		p := pick(r, procs)
		if level >= 3 && i%5 == 0 {
			p = "importer.exe"
			for h == host {
				h = pick(r, hosts)
			}
		}
		ts := fmt.Sprintf("2026-03-%02d %02d:%02d:%02d", 1+r.IntN(days), r.IntN(24), r.IntN(60), r.IntN(60))
		c := p + " /quiet"
		if p == "importer.exe" {
			c = fmt.Sprintf("importer.exe /db:hr /login:%s /pwd:Pw%04d!x", pick(r, logins), r.IntN(10000))
		}
		rows = append(rows, row{ts, h, pick(r, runAccounts), p, c})
	}
	if level >= 4 {
		r.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
	} else {
		sort.Slice(rows, func(i, j int) bool { return rows[i].ts < rows[j].ts })
	}
	var sb strings.Builder
	sb.WriteString("time,host,account,process,command_line\n")
	for _, rw := range rows {
		fmt.Fprintf(&sb, "%s,%s,%s,%s,%s\n", rw.ts, rw.host, rw.acct, rw.proc, rw.cmd)
	}
	window := minTOD + "-" + maxTOD
	gold := fmt.Sprintf(`{"runs":%d,"first":"%s","window":"%s","password_login":"%s"}`, runs, first, window, login)

	prompt := fmt.Sprintf("Process-start records (CSV, one row per start):\n\n%s\n"+
		"Answer from the records only, about `importer.exe` on host `%s`:\n"+
		"- runs: how many times did it start?\n"+
		"- first: the timestamp of its first start, exactly as in the records\n"+
		"- window: earliest and latest time of day it started, as HH:MM-HH:MM\n"+
		"- password_login: the password in its command line belongs to which login name?\n\n"+
		"%s The answer is one JSON object on a single line with the keys runs, first, window, password_login.",
		sb.String(), host, answerRule)

	check := func(resp string) (bool, string, bool) {
		var got struct {
			Runs          any    `json:"runs"`
			First         string `json:"first"`
			Window        string `json:"window"`
			PasswordLogin string `json:"password_login"`
		}
		raw, found, ok := answerJSON(resp, &got)
		if !found || !ok {
			return false, raw, found
		}
		runsOK := digitsOnly(fmt.Sprint(got.Runs)) == strconv.Itoa(runs)
		win := strings.ReplaceAll(strings.ReplaceAll(got.Window, " ", ""), "–", "-")
		correct := runsOK && strings.TrimSpace(got.First) == first && win == window &&
			strings.EqualFold(strings.TrimSpace(got.PasswordLogin), login)
		return correct, raw, true
	}
	return Task{Turns: user(prompt), Gold: gold, check: check}
}

// genSuperseded: which value is current after corrections, rejected proposals
// and out-of-order entries ("these are wrong numbers.. from before").
func genSuperseded(r *rand.Rand, level int) Task {
	type fact struct {
		key, label string
		values     []string
	}
	facts := []fact{
		{"memory_gib", "memory limit (GiB)", []string{"64", "72", "96", "108", "120"}},
		{"index", "active index", []string{"gen2", "gen3", "gen4", "gen5"}},
		{"workers", "worker count", []string{"2", "4", "6", "8", "12", "16"}},
		{"dataset", "dataset in use", []string{"full-export", "reduced-r01", "masked-r02", "twelve-day"}},
	}
	type entry struct {
		minute int
		text   string
	}
	var entries []entry
	current := map[string]string{}
	minute := 8 * 60
	for _, f := range facts {
		v := pick(r, f.values)
		current[f.key] = v
		entries = append(entries, entry{minute, fmt.Sprintf("Setup: %s is %s.", f.label, v)})
		minute += 3 + r.IntN(10)
	}
	nUpdates := 2 + 2*level
	for i := 0; i < nUpdates; i++ {
		f := facts[r.IntN(len(facts))]
		v := pick(r, f.values)
		for v == current[f.key] {
			v = pick(r, f.values)
		}
		minute += 5 + r.IntN(40)
		switch {
		case level >= 2 && r.IntN(3) == 0:
			entries = append(entries, entry{minute, fmt.Sprintf("Proposal: change the %s to %s. Rejected by the lead, nothing changed.", f.label, v)})
		case level >= 3 && r.IntN(4) == 0:
			entries = append(entries, entry{minute, fmt.Sprintf("Measured during the test run: the %s looked like %s in the logs, but the setting itself was not changed.", f.label, v)})
		default:
			entries = append(entries, entry{minute, fmt.Sprintf("Update: %s changed from %s to %s.", f.label, current[f.key], v)})
			current[f.key] = v
		}
	}
	ordered := "in chronological order"
	if level >= 4 {
		r.Shuffle(len(entries), func(i, j int) { entries[i], entries[j] = entries[j], entries[i] })
		ordered = "NOT in chronological order; the timestamp decides which entry is newer"
	}
	var sb strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&sb, "[%02d:%02d] %s\n", e.minute/60, e.minute%60, e.text)
	}
	ask := r.Perm(len(facts))[:3]
	sort.Ints(ask)
	var keys []string
	goldParts := []string{}
	for _, i := range ask {
		keys = append(keys, facts[i].key)
		goldParts = append(goldParts, fmt.Sprintf("%q:%q", facts[i].key, current[facts[i].key]))
	}
	gold := "{" + strings.Join(goldParts, ",") + "}"
	prompt := fmt.Sprintf("Handover notes from a work session. The entries are %s.\n\n%s\n"+
		"What are the CURRENT values of %s? %s The answer is one JSON object on a single line with those keys and string values.",
		ordered, sb.String(), strings.Join(keys, ", "), answerRule)

	check := func(resp string) (bool, string, bool) {
		got := map[string]any{}
		raw, found, ok := answerJSON(resp, &got)
		if !found || !ok {
			return false, raw, found
		}
		for _, k := range keys {
			if strings.TrimSpace(fmt.Sprint(got[k])) != current[k] {
				return false, raw, true
			}
		}
		return true, raw, true
	}
	return Task{Turns: user(prompt), Gold: gold, check: check}
}

// genProgressState: which items are not finished? Often the answer is "none"
// ("could it be that all days are already checked and you did not recognize this?").
func genProgressState(r *rand.Rand, level int) Task {
	n := 6 + 3*level
	items := make([]string, n)
	for i := range items {
		items[i] = fmt.Sprintf("day-%02d", i+1)
	}
	allDone := r.IntN(10) < 3
	type ev struct {
		t    int
		text string
	}
	var evs []ev
	t := 8 * 60
	var notDone []string
	for _, it := range items {
		state := "done"
		if !allDone {
			switch r.IntN(6) {
			case 0:
				state = "running"
			case 1:
				state = "failed"
			case 2:
				if level >= 2 {
					state = "never"
				}
			}
		}
		if state != "done" {
			notDone = append(notDone, it)
		}
		if state == "never" {
			continue
		}
		t += 2 + r.IntN(15)
		evs = append(evs, ev{t, it + " check started"})
		if level >= 2 && r.IntN(3) == 0 { // fail then retry
			t += 1 + r.IntN(10)
			evs = append(evs, ev{t, it + " check FAILED (out of memory)"})
			if state == "failed" {
				continue
			}
			t += 1 + r.IntN(5)
			evs = append(evs, ev{t, it + " check restarted"})
		}
		switch state {
		case "done":
			t += 1 + r.IntN(20)
			evs = append(evs, ev{t, it + " check finished: agree"})
			if level >= 3 && r.IntN(4) == 0 {
				t += 1 + r.IntN(10)
				evs = append(evs, ev{t, it + " recheck started by second driver; first result kept"})
			}
		case "failed":
			t += 1 + r.IntN(10)
			evs = append(evs, ev{t, it + " check FAILED (timeout), not retried"})
		}
	}
	for i := 0; i < level*2; i++ { // infrastructure noise unrelated to item results
		evs = append(evs, ev{8*60 + r.IntN(t-8*60+1), pick(r, []string{
			"driver copy step failed (12 files), replacement driver started",
			"watcher re-armed",
			"disk usage 81%",
			"driver copy step retried: ok",
		})})
	}
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].t < evs[j].t })
	var sb strings.Builder
	for _, e := range evs {
		fmt.Fprintf(&sb, "%02d:%02d %s\n", e.t/60, e.t%60, e.text)
	}
	gold := "none"
	if len(notDone) > 0 {
		gold = strings.Join(notDone, ",")
	}
	prompt := fmt.Sprintf("A verification sweep covers these items: %s.\nAn item is finished only when its check finished. Here is the full sweep log:\n\n%s\n"+
		"Which items are NOT finished yet? %s The answer is a comma-separated list of item names, or `none` if every item is finished.",
		strings.Join(items, ", "), sb.String(), answerRule)

	check := func(resp string) (bool, string, bool) {
		got, found := answerLine(resp)
		if !found {
			return false, "", false
		}
		if gold == "none" {
			return strings.EqualFold(strings.Trim(got, ". "), "none"), got, true
		}
		var have []string
		for _, f := range strings.FieldsFunc(got, func(c rune) bool { return c == ',' || c == ' ' || c == ';' }) {
			if strings.HasPrefix(f, "day-") {
				have = append(have, strings.Trim(f, ".`"))
			}
		}
		return strings.Join(sortedCopy(have), ",") == strings.Join(sortedCopy(notDone), ","), got, true
	}
	return Task{Turns: user(prompt), Gold: gold, check: check}
}

// genCLIFlags: read the help text, especially defaults, before running a tool
// (the tool silently redid the whole job 12 times because of a default).
func genCLIFlags(r *rand.Rand, level int) Task {
	passesDefault := 6 + r.IntN(10)
	in := fmt.Sprintf("data/day-%02d.parquet", 1+r.IntN(28))
	out := fmt.Sprintf("out/result-%d.json", r.IntN(1000))
	mem := pick(r, []int{32, 48, 64, 96})
	workers := pick(r, []int{2, 4, 8, 16})
	spill := pick(r, []string{"/fast/spill", "/scratch/tmp", "/nvme/spill"})

	help := fmt.Sprintf(`Usage: derive [flags]

Flags use Go syntax: -name value or -name=value. Boolean flags must be written
as -name=false to turn them off; "-name false" does NOT work.

  -in string        input file (required)
  -out string       output file (default "result.csv")
  -format string    output format: csv or json (default "csv")
  -passes int       TOTAL number of full passes over the input (minimum 1). Each
                    pass processes the whole input again and writes a checkpoint
                    (default %d)
  -verify           after finishing, run the whole job a second time and compare
                    the results (default true)
  -mem-limit string memory cap such as 64G; 0 means unlimited (default "0")
  -workers int      parallel workers (default 1)
  -spill-dir string directory for spilled data when the memory cap is reached
                    (default: none, the job fails when the cap is reached)
  -dry-run          print the plan and exit
`, passesDefault)

	type req struct {
		desc, frag string
		ok         func(map[string]string) bool
	}
	reqs := []req{
		{fmt.Sprintf("read %s", in), "-in " + in, func(f map[string]string) bool { return f["in"] == in }},
		{"process the input exactly once (no extra checkpoint passes)", "-passes 1", func(f map[string]string) bool { return f["passes"] == "1" }},
		{fmt.Sprintf("write JSON to %s", out), "-out " + out + " -format json", func(f map[string]string) bool { return f["out"] == out && f["format"] == "json" }},
		{"do not run the job a second time for verification", "-verify=false", func(f map[string]string) bool { return f["verify"] == "false" }},
		{fmt.Sprintf("cap memory at %d GiB", mem), fmt.Sprintf("-mem-limit %dG", mem), func(f map[string]string) bool {
			v := strings.TrimSuffix(strings.TrimSuffix(strings.ToUpper(f["mem-limit"]), "IB"), "B")
			return v == fmt.Sprintf("%dG", mem)
		}},
		{fmt.Sprintf("use %d workers", workers), fmt.Sprintf("-workers %d", workers), func(f map[string]string) bool { return f["workers"] == strconv.Itoa(workers) }},
		{fmt.Sprintf("spill to %s instead of failing when the cap is reached", spill), "-spill-dir " + spill, func(f map[string]string) bool { return f["spill-dir"] == spill }},
	}
	nReq := min(len(reqs), 2+level)
	reqs = reqs[:nReq]
	var goal []string
	cmd := []string{"derive"}
	for _, rq := range reqs {
		goal = append(goal, rq.desc)
		cmd = append(cmd, rq.frag)
	}
	prompt := fmt.Sprintf("Here is the help text of a tool:\n\n```\n%s```\n\nWrite the single command line that does the following: %s. "+
		"Set only what is needed. %s The answer is the command on one line, starting with `derive`.",
		help, strings.Join(goal, "; "), answerRule)

	gold := strings.Join(cmd, " ")
	check := func(resp string) (bool, string, bool) {
		got, found := answerLine(resp)
		if !found {
			return false, "", false
		}
		flags := parseFlags(got)
		if flags == nil {
			return false, got, true
		}
		for _, rq := range reqs {
			if !rq.ok(flags) {
				return false, got, true
			}
		}
		if _, dry := flags["dry-run"]; dry && flags["dry-run"] != "false" {
			return false, got, true
		}
		return true, got, true
	}
	return Task{Turns: user(prompt), Gold: gold, check: check}
}

var boolFlags = map[string]bool{"verify": true, "dry-run": true}

// parseFlags parses "derive -a x -b=y" with Go flag semantics: a bare boolean
// flag means true, and "-verify false" leaves verify=true plus a stray argument.
// Returns nil when the line does not start with derive or has stray arguments.
func parseFlags(line string) map[string]string {
	toks := strings.Fields(line)
	if len(toks) == 0 || toks[0] != "derive" {
		return nil
	}
	out := map[string]string{}
	for i := 1; i < len(toks); i++ {
		tok := toks[i]
		if !strings.HasPrefix(tok, "-") {
			return nil // positional argument: Go's flag package stops parsing here
		}
		name := strings.TrimLeft(tok, "-")
		if k, v, ok := strings.Cut(name, "="); ok {
			out[k] = strings.Trim(v, `"'`)
			continue
		}
		if boolFlags[name] {
			out[name] = "true"
			continue
		}
		if i+1 >= len(toks) {
			return nil
		}
		out[name] = strings.Trim(toks[i+1], `"'`)
		i++
	}
	return out
}

// genCostChoice: pick the cheapest option that meets every requirement
// ("very stupid and expensive call": a 75-minute rerun to change one line).
func genCostChoice(r *rand.Rand, level int) Task {
	featPool := []string{
		"the page names the data source",
		"the hourly chart uses the corrected timezone",
		"the summary table has the new column",
		"the numbers match the raw records",
		"the German page is regenerated",
		"the old totals stay unchanged",
	}
	nReq := 2 + level/2
	required := r.Perm(len(featPool))[:nReq]
	reqSet := map[int]bool{}
	for _, i := range required {
		reqSet[i] = true
	}
	type option struct {
		desc   string
		covers map[int]bool
		cost   int // seconds
	}
	var opts []option
	all := func() map[int]bool {
		m := map[int]bool{}
		for i := range featPool {
			m[i] = true
		}
		return m
	}
	// The correct option: covers exactly what is needed, cheaply.
	best := 30 + r.IntN(600)
	opts = append(opts, option{"re-render the pages from the existing data file and patch the label", reqSet, best})
	// Expensive option that covers everything.
	opts = append(opts, option{"re-run the full report pipeline over all twelve days", all(), best*4 + 3600 + r.IntN(3600)})
	// Cheaper options that each miss one requirement.
	// Neutral names, so the description never contradicts the delivery list.
	nearMiss := []string{"run the quick patch script", "run the partial rebuild", "apply the hotfix template", "run the incremental update"}
	for i := 0; i < 1+level/2; i++ {
		m := map[int]bool{}
		for k := range reqSet {
			m[k] = true
		}
		delete(m, required[r.IntN(len(required))])
		opts = append(opts, option{nearMiss[i%len(nearMiss)], m, 5 + r.IntN(best)})
	}
	// A middle option that covers all but costs more than the best.
	opts = append(opts, option{"re-run the report for the affected day, then re-render", all(), best + 300 + r.IntN(1800)})

	fmtCost := func(s int) string {
		if level < 3 {
			return fmt.Sprintf("%d min", (s+59)/60)
		}
		switch r.IntN(3) {
		case 0:
			return fmt.Sprintf("%d s", s)
		case 1:
			return fmt.Sprintf("%.1f min", float64(s)/60)
		default:
			return fmt.Sprintf("%.2f h", float64(s)/3600)
		}
	}
	idx := r.Perm(len(opts))
	var sb strings.Builder
	gold := ""
	for pos, i := range idx {
		o := opts[i]
		var cov []string
		for k := range featPool {
			if o.covers[k] {
				cov = append(cov, featPool[k])
			}
		}
		l := string(rune('A' + pos))
		fmt.Fprintf(&sb, "%s) %s. Cost: %s. Delivers: %s.\n", l, o.desc, fmtCost(o.cost), strings.Join(cov, "; "))
		if i == 0 {
			gold = l
		}
	}
	var need []string
	for _, i := range required {
		need = append(need, featPool[i])
	}
	prompt := fmt.Sprintf("The user asked for a small change to an already finished report. Requirements:\n- %s\n\nOptions:\n%s\n"+
		"Choose the cheapest option that meets ALL requirements. %s The answer is the option letter.",
		strings.Join(need, "\n- "), sb.String(), answerRule)
	return Task{Turns: user(prompt), Gold: gold, check: checkLetter(gold)}
}
