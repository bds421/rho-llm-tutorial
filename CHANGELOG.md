# Changelog

## v0.3.2 — 2026-03-24

### Changed
- Bump `rho/llm` dependency from v0.2.1 to v0.2.3 across all 22 modules + root
- v0.2.2 added `ThinkingBudgetTokens`, per-request `ThinkingBudget`, `ReasoningSummary`, new `ThinkingMinimal`/`ThinkingXHigh` levels
- v0.2.3 fixed Gemini adapter sending `thinkingConfig` at wrong JSON level (top-level instead of inside `generationConfig`)

### Added — Tutorial 04 (Thinking) extensions
- `thinking_budget_test.go` — live integration tests for ThinkingBudget across all three providers:
  - **Anthropic** (`claude-haiku-4-5-20251001`): per-request `ThinkingLevel` + `ThinkingBudget` override
  - **Gemini** (`gemini-2.5-flash`): native thinking verification (explicit `ThinkingLevel` was broken in v0.2.2, fixed in v0.2.3)
  - **OpenAI** (`gpt-5.4-nano`): `ThinkingLevel` + `ReasoningSummaryDetailed` via Responses API
- `TestThinkingBudgetTokensDefaults` — unit test for `ThinkingBudgetTokens()` helper across all levels (minimal→1024, low→4096, medium→16384, high→65536, xhigh→128000) plus custom override
- `TestReasoningSummaryConstants` — verifies `ReasoningSummaryNone`/`Auto`/`Detailed`/`Concise` values
- Steps 4–6 in `main.go`: `ThinkingBudgetTokens` demo, per-request `ThinkingBudget` override, `ReasoningSummary` constants

### Added — Tutorial 22 (Cloud CTL HTTP Tool Use)
- New tutorial module with HTTP-based tool use tests

### Improved — CLAUDE.md
- Document `.env` file for API keys and testing workflow (start small, avoid full IQ test)

## v0.3.1 — 2026-03-18

### Added — Tutorial 18 (Content Model) live vision tests
- `TestGeminiVisionDescribesImage` — sends a real image to Gemini 2.5 Flash, verifies it identifies a golden retriever (skipped without `GEMINI_API_KEY`)
- `TestAnthropicVisionDescribesImage` — sends the same image to Anthropic Haiku, verifies the response (skipped without `ANTHROPIC_API_KEY`)
- Test image: golden retriever cropped from [robert-mcdermott/LLM-Image-Classification](https://github.com/robert-mcdermott/LLM-Image-Classification) dataset (`testdata/dog.png`)

## v0.3.0 — 2026-03-18

### Changed — **Breaking** (rho/llm v0.2.1)
- Bump `rho/llm` dependency from v0.1.22 to v0.2.1 across all 22 modules
- **`Config.Temperature`** changed from `float64` to `*float64` — `nil` means "use provider default, don't send on wire". Updated tutorials 08 and 12.
- **`EstimateCost`** replaced with `EstimateCost(CostInput)` — accepts `ThinkingTokens`, `CacheCreateTokens`, `CacheReadTokens` for accurate pricing. Updated tutorials 06, 08, 15, 18.
- **`Config.Temperature` display** in tutorial 13 now handles nil pointer

## v0.2.9 — 2026-03-18

### Improved — Tutorial 15 (Multi-Provider Comparison)
- Document token budget tradeoff for reasoning models — models with `info.Thinking == true` consume output tokens for chain-of-thought before producing visible answers
- Increase Ollama Qwen3 `MaxTokens` from 1024 to 2048 for reliable answers on longer prompts
- Show `[thinks]` registry tag in streaming output via `llm.GetModelInfo`
- Unify `MaxTokens` to 100 for non-reasoning providers (Gemini, Anthropic)
- Bump `rho/llm` dependency to v0.1.22 — exposes `ThinkingTokens` on `llm.Response`

### Added — Tutorial 15 integration tests
- `TestGeminiThinkingModelReturnsContent` — verifies Gemini adapter's `maxOutputTokens` padding produces non-empty content (skipped without `GEMINI_API_KEY`)
- `TestOllamaThinkingModelReturnsContent` — verifies Ollama reasoning field parsed into `resp.Thinking` and content is non-empty with sufficient budget (skipped without Ollama)

## v0.2.8 — 2026-03-18

### Added — Capability test questions (tutorial 20)
- **Level 4**: `logic-alpha-sequence` — digits sorted alphabetically by English name (→ 0)
- **Level 5**: `advanced-logic-grid` — 8-constraint multi-variable deduction (5 scientists, 5 elements, 5 floors)
- **Level 5**: `advanced-calculator` — minimum button presses ×2/+1 to reach 100 (→ 8, work-backwards strategy)
- **Level 5**: `advanced-self-referential` — self-describing 10-digit sequence (→ 6210001000)
- **Level 5**: `advanced-epistemic` — three-logician epistemic reasoning with elimination (→ 3)
- **Level 5**: `advanced-clock-trisection` — impossibility proof: three hands never trisect (→ 0)

### Improved — Tutorial 18 (Content Model)
- Add live vision API demo (Step 8): sends base64 image to Gemini, displays response + cost
- Demonstrate `NewImageMessage` helper and `ValidateImageSource` (Step 7)
- Bump `rho/llm` dependency from v0.1.20 to v0.1.21 — adds image/vision support for all 3 adapters

### Fixed
- **Validator logic**: `not_expected` now only fires when `expected` is absent — correct answers with intermediate work (e.g., showing `1/8` en route to `1/7`) no longer cause false failures

## v0.2.7 — 2026-03-18

### Changed
- Bump `rho/llm` dependency from v0.1.18 to v0.1.20 — adds thinking/reasoning content parsing for Gemini and OpenAI-compat providers

### Improved — Tutorial 15 (Multi-Provider Comparison)
- Display `resp.Thinking` content for models that reason by default (Gemini 2.5 Flash, Ollama Qwen3)
- Add `EstimateCost` to streaming comparison output
- Handle `EventThinking` stream events, show accumulated thinking size
- Increase Ollama Qwen3 `MaxTokens` from 50 to 1024 — reasoning models need headroom for chain-of-thought + answer
- Update tutorial header to mention thinking/reasoning content

## v0.2.6 — 2026-03-18

### Added
- **20_capability_test**: `-config` flag to specify a custom model config YAML (`go test -config config_run.yaml`)
- **20_capability_test**: `-short` mode runs English only, skipping DE/ES (3× faster)

### Changed — Capability test questions (tutorial 20)
- **Level 3** (Cognitive Reflection): Replace "switches" lateral-thinking riddle with "pills" CRT trap (intuitive 90 vs correct 60)
- **Level 4** (Multi-Step Deduction): Replace 4 internet puzzles (hats, knights/knaves, two-doors, 12-coins) with genuinely harder problems:
  - `logic-speed`: Harmonic mean trap (60/40 km/h → 48, not 50)
  - `logic-digit`: Three-digit constraint-satisfaction algebra (→ 194)
  - `logic-calendar`: 4-step temporal chain deduction (→ Sunday)
  - `logic-weighing`: 9-coin balance derivation (→ 2 weighings)
- **Level 5** (Advanced Reasoning): Replace 3 trivial/famous puzzles (pigeonhole, set intersection, snail) with:
  - `advanced-clock`: Continuous hour-hand motion at 3:15 (→ 7.5°)
  - `advanced-hanoi`: Tower of Hanoi recursive formula for 6 disks (→ 63)
  - `advanced-derangement`: Inclusion-exclusion probability for 4 letters (→ 3/8)
- Rename categories: "Math/IQ" → "Cognitive Reflection", "Logic/IQ/Mensa" → "Multi-Step Deduction" / "Advanced Reasoning"

## v0.2.5 — 2026-03-18

### Changed
- Replace short model aliases with full registry IDs in 12 tutorial Go files (`"flash"` → `"gemini-2.5-flash"`, `"haiku"` → `"claude-haiku-4-5-20251001"`, `"sonnet"` → `"claude-sonnet-4-6"`)
- Remove alias-related inline comments (e.g. `// alias — resolves to ...`)
- Remove "Short aliases … are also accepted" comment from config headers in tutorials 20 and 21

### Added
- **CLAUDE.md**: Rule — always use full model IDs, never short aliases (except tutorial 06 which demos `ResolveModelAlias`)

## v0.2.4 — 2026-03-18

### Changed
- Bump `rho/llm` dependency from v0.1.17 to v0.1.18 (all 22 modules) — fixes Mistral `max_completion_tokens` rejection (HTTP 422) for `mistral-small-2603`

### Fixed
- **21_cloud_ctl_tool_use/tests.yaml**: Add missing `"no matches"` to `empty_keywords` in `error-empty-search` test case — was causing false failures for models using that phrasing
- **.gitignore**: Add `*/config_*.yaml` pattern to exclude ad-hoc test configs from tracking

## v0.2.3 — 2026-03-18

### Changed
- Bump `rho/llm` dependency from v0.1.16 to v0.1.17 (all 22 modules)
- Bump Go directive from 1.26.0 to 1.26.1 (all 22 modules)
- Switch git origin from GitLab to GitHub; remove redundant `github` remote

### Added — New models in test configs
- **20_capability_test/config.yaml**:
  - Active: `gemini-3.1-flash-lite-preview`, `grok-4-fast-non-reasoning`, `mistral-small-2603`
  - Commented references: `gemini-2.0-flash`, `grok-4.20-beta`, `grok-4-fast-reasoning`, `grok-4-0709`, `gpt-5.4-nano`, `gpt-4.1-nano`, `llama-3.3-70b-versatile`, `deepseek-r1-distill-llama-70b`, `mistral-medium-latest`, `codestral-2508`, `devstral-2512`, `magistral-medium-2509`
  - New sections: OpenAI, Groq
- **21_cloud_ctl_tool_use/config.yaml**:
  - Active: `mistral-small-2603`
  - Commented references: `grok-4.20-beta-0309-non-reasoning`, `gpt-5.4-nano`, `mistral-small-2603`

### Added — Tutorial 06 updates
- New aliases demoed: `gpt`, `mistral-small`, `groq`, `codestral`
- New models in info queries: `gpt-5.4-nano`, `mistral-small-2603`
- New providers in default model section: `groq`, `mistral`

### Fixed
- **08_auth_pool_failover/main.go**: Replace stale `gpt-4o` with `gpt-4.1` (gpt-4o removed from registry)
- **06_cost_and_registry/main.go**: Replace stale `gpt-5.2` with `gpt-5.4` in provider detection

### Fixed — Report directory convention
- Change `report_dir` from `testdata` to `reports` in tutorials 20 and 21
- Generated test reports are no longer checked into git (`*/reports/` added to `.gitignore`)

## v0.2.2 — 2026-03-18

- GitHub migration cleanup for bds421/rho-llm-tutorial

## v0.2.1

- Remove `cl` dependency from tutorial 21, add parallelization
