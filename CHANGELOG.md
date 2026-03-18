# Changelog

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
