# Tutorial upgrade to rho/llm v0.2.7 (v0.3.3)

## Completed

- [x] `git pull` — sync with remote v0.3.2 (v0.2.3 + ThinkingBudget + tutorial 22)
- [x] `go get github.com/bds421/rho-llm@v0.2.7` — all 23 modules + root
- [x] `go mod tidy` — all modules clean
- [x] `go build ./...` — all tutorials compile
- [x] `go vet ./...` — no issues
- [x] Add Gemma 4 models (`gemma4:e4b`, `gemma4:26b`, `gemma4:31b`) to `20_capability_test/config.yaml`
- [x] Add Gemma 4 models to `21_cloud_ctl_tool_use/config.yaml` (tool use support is new in Gemma 4)
- [x] Update root `main.go` demo from `gemma3:4b` → `gemma4:e4b`
- [x] Add `dashscope` and `ollama` to `13_registry_deep/main.go` provider list
- [x] Add `git pull first` rule to CLAUDE.md
- [x] ThinkingBudget unit tests pass with v0.2.7 (tutorial 04)
- [x] Stress tests pass with v0.2.7 (tutorial 19, mock provider)
- [x] Capability test — Gemma 4 report generated: e4b 75.3%, 26b 79.6%, 31b 100% (partial)
- [x] Tool use test — Gemma 4 report generated: e4b 93.3%, 26b 100%, 31b 100%
- [x] No regressions vs previous v0.2.3 run
- [x] Update CHANGELOG.md (v0.3.3)
- [x] Commit, tag v0.3.3, push
