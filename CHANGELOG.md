# Changelog

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
