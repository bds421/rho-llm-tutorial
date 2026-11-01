package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/big"
	"math/rand/v2"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// =============================================================================
// Task model
// =============================================================================

// Turn is one message of the conversation sent to the model. The last turn is
// always a user turn.
type Turn struct {
	Role string // "user" or "assistant"
	Text string
}

// Task is one generated, exactly-checkable question.
type Task struct {
	ID     string
	Family string
	Level  int
	Turns  []Turn
	Gold   string
	// check grades the full model response. It returns whether the answer is
	// correct and the extracted answer (for the log). found is false when the
	// response has no ANSWER line at all.
	check func(resp string) (correct bool, extracted string, found bool)
}

// Check grades a response.
func (t Task) Check(resp string) (correct bool, extracted string, found bool) {
	return t.check(resp)
}

// generator builds one task at a level (1–5) from a deterministic RNG.
type generator func(r *rand.Rand, level int) Task

// Families is the registry of task families, in report order.
var Families = []struct {
	Name   string
	Group  string // "intent", "agentic" or "precision"
	Gen    generator
	Weight int // relative default count
}{
	{"intent", "intent", genIntent, 3},
	{"instruct_persist", "intent", genInstructPersist, 2},
	{"log_count", "agentic", genLogCount, 1},
	{"superseded", "agentic", genSuperseded, 1},
	{"progress_state", "agentic", genProgressState, 1},
	{"cli_flags", "agentic", genCLIFlags, 1},
	{"cost_choice", "agentic", genCostChoice, 1},
	{"arith_mul", "precision", genArithMul, 1},
	{"var_track", "precision", genVarTrack, 1},
	{"kv_recall", "precision", genKVRecall, 1},
	{"str_reverse", "precision", genStrReverse, 1},
}

func familyGen(name string) (generator, bool) {
	for _, f := range Families {
		if f.Name == name {
			return f.Gen, true
		}
	}
	return nil, false
}

// seededRand derives a deterministic RNG from any number of string parts.
func seededRand(parts ...string) *rand.Rand {
	h := fnv.New64a()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	s := h.Sum64()
	return rand.New(rand.NewPCG(s, s^0x9e3779b97f4a7c15))
}

// Generate builds task number idx of a family for a seed set. The same
// (set, family, level, idx) always yields the same task.
func Generate(set, family string, level, idx int) (Task, error) {
	gen, ok := familyGen(family)
	if !ok {
		return Task{}, fmt.Errorf("unknown family %q", family)
	}
	if level < 1 || level > 5 {
		return Task{}, fmt.Errorf("level %d out of range 1..5", level)
	}
	r := seededRand(set, family, strconv.Itoa(level), strconv.Itoa(idx))
	t := gen(r, level)
	t.ID = fmt.Sprintf("%s/L%d/%s/%03d", family, level, set, idx)
	t.Family = family
	t.Level = level
	return t, nil
}

const answerRule = "Finish your reply with one final line of the form `ANSWER: <answer>`."

// =============================================================================
// Answer extraction
// =============================================================================

var answerMarker = regexp.MustCompile(`(?i)answer\s*\**\s*:`)

// answerTail returns everything after the LAST "ANSWER:" marker, with markdown
// decoration trimmed. found is false when there is no marker.
func answerTail(resp string) (string, bool) {
	locs := answerMarker.FindAllStringIndex(resp, -1)
	if len(locs) == 0 {
		return "", false
	}
	tail := resp[locs[len(locs)-1][1]:]
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(tail), "*`")), true
}

// answerLine returns the first line after the last ANSWER marker.
func answerLine(resp string) (string, bool) {
	tail, ok := answerTail(resp)
	if !ok {
		return "", false
	}
	line, _, _ := strings.Cut(tail, "\n")
	return cleanToken(line), true
}

// cleanToken strips surrounding whitespace, markdown and quotes.
func cleanToken(s string) string {
	return strings.Trim(strings.TrimSpace(s), "*`\"' .")
}

// answerJSON decodes the first JSON object after the last ANSWER marker.
func answerJSON(resp string, v any) (string, bool, bool) {
	tail, ok := answerTail(resp)
	if !ok {
		return "", false, false
	}
	start := strings.Index(tail, "{")
	if start < 0 {
		return firstLine(tail), true, false
	}
	depth := 0
	for i := start; i < len(tail); i++ {
		switch tail[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				obj := tail[start : i+1]
				return obj, true, json.Unmarshal([]byte(obj), v) == nil
			}
		}
	}
	return firstLine(tail), true, false
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}

var nonDigit = regexp.MustCompile(`[^0-9-]`)

// checkLine compares the answer line against gold with a normalizer.
func checkLine(gold string, norm func(string) string) func(string) (bool, string, bool) {
	return func(resp string) (bool, string, bool) {
		got, ok := answerLine(resp)
		if !ok {
			return false, "", false
		}
		return norm(got) == norm(gold), got, true
	}
}

func digitsOnly(s string) string { return nonDigit.ReplaceAllString(s, "") }

// =============================================================================
// Precision families — sensitive to quantization and reduced reasoning effort
// =============================================================================

// genArithMul: exact multi-digit multiplication without tools.
func genArithMul(r *rand.Rand, level int) Task {
	digits := level + 4 // L1: 5×5 … L5: 9×9
	a, b := randDigits(r, digits), randDigits(r, digits)
	x, _ := new(big.Int).SetString(a, 10)
	y, _ := new(big.Int).SetString(b, 10)
	gold := new(big.Int).Mul(x, y).String()
	prompt := fmt.Sprintf("Compute %s × %s exactly. Do not use code or tools; work it out by hand. %s The answer must be the product as plain digits.",
		a, b, answerRule)
	return Task{Turns: user(prompt), Gold: gold, check: checkLine(gold, digitsOnly)}
}

func randDigits(r *rand.Rand, n int) string {
	var sb strings.Builder
	sb.WriteByte(byte('1' + r.IntN(9)))
	for i := 1; i < n; i++ {
		sb.WriteByte(byte('0' + r.IntN(10)))
	}
	return sb.String()
}

// genVarTrack: state tracking over many assignment/swap steps.
func genVarTrack(r *rand.Rand, level int) Task {
	nVars := 3 + level
	nOps := 6*level + 4
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h"}[:nVars]
	vals := map[string]int{}
	var sb strings.Builder
	step := 1
	for _, n := range names {
		v := r.IntN(20)
		vals[n] = v
		fmt.Fprintf(&sb, "%d. %s = %d\n", step, n, v)
		step++
	}
	for i := 0; i < nOps; i++ {
		x := names[r.IntN(nVars)]
		y := names[r.IntN(nVars)]
		for y == x {
			y = names[r.IntN(nVars)]
		}
		switch r.IntN(5) {
		case 0:
			k := 1 + r.IntN(9)
			vals[x] += k
			fmt.Fprintf(&sb, "%d. %s += %d\n", step, x, k)
		case 1:
			vals[x] -= vals[y]
			fmt.Fprintf(&sb, "%d. %s -= %s\n", step, x, y)
		case 2:
			vals[x], vals[y] = vals[y], vals[x]
			fmt.Fprintf(&sb, "%d. swap %s and %s\n", step, x, y)
		case 3:
			vals[x] = vals[y]
			fmt.Fprintf(&sb, "%d. %s = %s\n", step, x, y)
		default:
			vals[x] += vals[y]
			fmt.Fprintf(&sb, "%d. %s += %s\n", step, x, y)
		}
		step++
	}
	target := names[r.IntN(nVars)]
	gold := strconv.Itoa(vals[target])
	prompt := fmt.Sprintf("Execute these steps in order. All values are integers.\n\n%s\nWhat is the final value of %s? %s",
		sb.String(), target, answerRule)
	return Task{Turns: user(prompt), Gold: gold, check: checkLine(gold, digitsOnly)}
}

// genKVRecall: exact retrieval among many lookalike keys.
func genKVRecall(r *rand.Rand, level int) Task {
	n := 30 * level
	keys := map[string]bool{}
	var order []string
	vals := map[string]string{}
	add := func(k string) {
		if keys[k] {
			return
		}
		keys[k] = true
		order = append(order, k)
		vals[k] = strconv.Itoa(100000 + r.IntN(900000))
	}
	for len(order) < n {
		k := fmt.Sprintf("srv-%04x", r.IntN(0x10000))
		add(k)
		if level >= 3 && r.IntN(3) == 0 { // lookalike: one hex digit changed
			b := []byte(k)
			pos := 4 + r.IntN(4)
			b[pos] = "0123456789abcdef"[r.IntN(16)]
			add(string(b))
		}
	}
	r.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	var sb strings.Builder
	for _, k := range order {
		fmt.Fprintf(&sb, "%s = %s\n", k, vals[k])
	}
	ask := make([]string, 3)
	var gold []string
	for i := range ask {
		ask[i] = order[r.IntN(len(order))]
		gold = append(gold, vals[ask[i]])
	}
	g := strings.Join(gold, ",")
	prompt := fmt.Sprintf("Here is a table of keys and values.\n\n%s\nGive the values for %s, %s and %s, in that order, comma-separated. %s",
		sb.String(), ask[0], ask[1], ask[2], answerRule)
	norm := func(s string) string {
		return strings.Join(regexp.MustCompile(`\d{6}`).FindAllString(s, -1), ",")
	}
	return Task{Turns: user(prompt), Gold: g, check: checkLine(g, norm)}
}

// genStrReverse: exact character-level copy, reversed.
func genStrReverse(r *rand.Rand, level int) Task {
	const alpha = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	n := 20 + 30*level
	b := make([]byte, n)
	for i := range b {
		b[i] = alpha[r.IntN(len(alpha))]
	}
	s := string(b)
	rev := make([]byte, n)
	for i := range b {
		rev[i] = b[n-1-i]
	}
	gold := string(rev)
	prompt := fmt.Sprintf("Reverse this string character by character (case-sensitive):\n\n%s\n\n%s", s, answerRule)
	return Task{Turns: user(prompt), Gold: gold, check: checkLine(gold, func(s string) string { return s })}
}

// =============================================================================
// Helpers shared by generators
// =============================================================================

func user(text string) []Turn { return []Turn{{Role: "user", Text: text}} }

func pick[T any](r *rand.Rand, xs []T) T { return xs[r.IntN(len(xs))] }

func sortedCopy(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	return out
}

// letters assigns A, B, C … to options after shuffling and returns the letter
// of the option at index correct.
func letterOptions(r *rand.Rand, options []string, correct int) (text string, gold string) {
	idx := r.Perm(len(options))
	var sb strings.Builder
	for pos, i := range idx {
		l := string(rune('A' + pos))
		fmt.Fprintf(&sb, "%s) %s\n", l, options[i])
		if i == correct {
			gold = l
		}
	}
	return sb.String(), gold
}

var letterRe = regexp.MustCompile(`\b([A-H])\b`)

// checkLetter accepts "B", "B)", "**B**", "Option B".
func checkLetter(gold string) func(string) (bool, string, bool) {
	return func(resp string) (bool, string, bool) {
		line, ok := answerLine(resp)
		if !ok {
			return false, "", false
		}
		if len(strings.Trim(line, ")")) == 1 {
			line = strings.ToUpper(line)
		}
		m := letterRe.FindStringSubmatch(line)
		if m == nil {
			return false, line, true
		}
		return m[1] == gold, line, true
	}
}
