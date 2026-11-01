# Tutorial 25 — Drift Canary

Has a model API become worse? This tool answers that with numbers instead of
a feeling: it sends freshly generated, exactly checkable tasks to each model,
logs the answers together with the response metadata, and compares a run
against a baseline with paired statistics.

## Why not an IQ test?

Classic puzzles (bat and ball, lily pads, Tower of Hanoi) do not show a
degraded model: every model has memorized them, quantized or not. And a
single pass/fail answer per question is too noisy to tell 80% from 70%.

What changes when a model is nerfed:

| What changed | What it looks like | Caught by |
|---|---|---|
| Less reasoning effort, smaller thinking budget | fewer output/thinking tokens, multi-step tasks fail | token comparison, `agentic` and `precision` groups |
| Quantization (FP8/INT4) | exact precision breaks first: digits, copying, long state tracking, recall among lookalikes | `precision` group |
| Routing to a smaller or different model | everything drops a little, `served_model` or latency changes | all groups, served-model column |
| Worse understanding of terse instructions | wrong action picked, rules from earlier turns ignored | `intent` group |
| More safety refusals | `stop_reason: refusal` on harmless tasks | refusal column |

## Task families

All tasks are generated from a seed, so they cannot be memorized, and every
answer is checked exactly. Each reply must end with `ANSWER: …`; only the last
such line counts.

**Intent: does the model understand the user?** (weighted highest)

- `intent`: a terse, typo-laden, often German/English-mixed user message
  after an assistant turn, and four possible next actions. The scenarios and
  the wrong options are distilled from real agent sessions where the model
  misread the user: doing the deferred step now ("the quick check is for the
  next step"), restoring its own style when told to keep the current one,
  defending stale numbers, asking a question the user already answered,
  finishing an expensive run when told to stop and build a POC, editing the
  wrong draft, implementing instead of writing a TODO, and reusing the old
  data source.
- `instruct_persist`: rules set in turn 1 (answer in English, start with
  `GOAL:`, no lists or tables) must still hold several turns later, when the
  user writes in German. The question contains a fencepost trap.

**Agentic: everyday analysis work** (all data synthetic)

- `log_count`: count the runs of one job in CSV process records, give its
  first start and time window, and say which login the password in the
  command line belongs to. The `account` column is the trap.
- `superseded`: current values from handover notes with corrections,
  rejected proposals and out-of-order timestamps.
- `progress_state`: which items of a sweep are not finished? 30% of the time
  the answer is `none`.
- `cli_flags`: write a command from a help text. The defaults matter (a
  default of several full passes) and so does Go flag syntax
  (`-verify=false`, not `-verify false`).
- `cost_choice`: the cheapest option that meets all requirements, with costs
  in mixed units.

**Precision: sensitive to quantization**

- `arith_mul`: multi-digit multiplication by hand
- `var_track`: variable values after many assignment and swap steps
- `kv_recall`: exact values among many lookalike keys
- `str_reverse`: reverse a random string character by character

Inspect any task:

```bash
go run . show -family intent -level 4 -idx 0
```

## Run

```bash
set -a; source ../.env; set +a

go run . run -set fixed                    # same tasks every time → paired comparison
go run . run -set fresh                    # new unseen tasks (guards against tuning to the test)
go run . run -set fixed -count 2 -models opus-5-5 -families intent,log_count
go run . summary runs/<run>.jsonl
go run . compare -base runs/<baseline>.jsonl -cand runs/<new>.jsonl
```

`compare` exits with status 3 when any cell shows a significant drop, so it
can raise an alarm from cron or CI.

Each run writes `runs/<timestamp>.jsonl` (git-ignored), one line per answer:
task, outcome (`pass`, `fail`, `no_answer`, `truncated`, `refusal`, `error`),
extracted and expected answer, served model, input/output/thinking tokens,
stop reason and latency.

## Getting a useful signal

1. **Pin everything** in `config.yaml`: model ID, thinking level, temperature
   (where the model accepts it), `max_tokens`, levels and count. Any change to
   the config is a change in the experiment, not in the provider.
2. **Calibrate the levels.** A family where the healthy model scores 100% or
   0% cannot show a drop. Aim for about 60–80% per family by raising or
   lowering its level in `config.yaml`.
3. **Take a baseline** on the `fixed` set when the model works well, and keep
   that JSONL file.
4. **Repeat on a schedule** (for example daily, at different times of day):
   degradations reported in practice were often intermittent and affected
   only part of the traffic.
5. **Use enough tasks.** On the fixed set, `compare` uses an exact McNemar
   test on the tasks both runs answered; otherwise Fisher's exact test. A
   10-point drop from 80% needs roughly 150–200 tasks per model to be
   significant, which is `count: 10` (160 tasks).

Errors and refusals do not count against accuracy. They are reported
separately, and a significant rise in refusals is flagged.

## Models not yet in the rho-llm registry

rho-llm refuses models without reviewed capability metadata. Declare them in
`config.yaml` until the registry includes them:

```yaml
  - provider: "anthropic"
    model: "claude-opus-5-5"
    api_key_env: "ANTHROPIC_API_KEY"
    capabilities: [chat, stream, tools, reasoning]
```

## Limits

- This tests the API model. If an agent such as a coding CLI behaves worse,
  the cause can also be its system prompt, context handling or tools, which
  this does not measure.
- Multiple-choice intent tasks measure understanding, not the action itself.
  A model that picks the right option can still act wrongly in a long session.
- Temperature 0 is not fully deterministic on most providers. Treat single
  flipped answers as noise and rely on the statistics.

## Offline tests

```bash
go test -race ./...
```

These need no API key. They check that every generated task accepts its own
gold answer at every level, and that near misses and trap answers are rejected
(an off-by-one digit, the run account as password owner, `-verify false`,
`none` when items are unfinished, a German reply). They also check the
statistics against known values.
