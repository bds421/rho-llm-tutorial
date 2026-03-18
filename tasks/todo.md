# Tutorial migration to rho/llm v0.2.1

## Breaking changes to apply

### 1. Temperature: float64 → *float64
- [ ] `08_auth_pool_failover/main.go:36` — `Temperature: 0.7` → `Temperature: llm.Float64(0.7)` or `ptr(0.7)`
- [ ] `12_request_overrides/main.go:26` — `Temperature: 1.0` → pointer
- [ ] `12_request_overrides/main.go:46` — `Temperature: temp` (Request loop) → pointer
- [ ] `13_registry_deep/main.go:24` — `cfg.Temperature` read — check if print format needs deref

### 2. EstimateCost → EstimateCostDetailed (no backward compat)
- [ ] `06_cost_and_registry/main.go:71` — `llm.EstimateCost(s.model, s.input, s.output)`
- [ ] `08_auth_pool_failover/main.go:63` — `llm.EstimateCost("claude-sonnet-4-6", ...)`
- [ ] `15_multi_provider/main.go:116` — `llm.EstimateCost(resolvedModel, ...)`
- [ ] `15_multi_provider/main.go:173` — `llm.EstimateCost(resolvedModel, ...)` (streaming)
- [ ] `18_content_model/main.go:222` — `llm.EstimateCost(cfg.Model, ...)`

### 3. Unknown providers require BaseURL
- [ ] `08_auth_pool_failover/main.go:115` — `Provider: "custom"` already has BaseURL ✓ (verify still works)
- [ ] `14_provider_helpers/main.go:52` — `Provider: "custom"` in preset demo — may need adjustment
- [ ] `16_pool_deep_dive/main.go:141` — `Provider: "demo"` — mock, verify RegisterProvider fires before validation
- [ ] `19_stress_tests/*` — 12 callsites with `Provider: "mock"` — uses RegisterProvider, verify order

### 4. Post-migration
- [ ] `go get github.com/bds421/rho-llm@v0.2.1` on root + all 21 submodules
- [ ] `make build-all` passes
- [ ] Run tutorial 15 live (Gemini + Ollama + Anthropic)
- [ ] Run `15_multi_provider` integration tests
- [ ] Run `19_stress_tests` (mock provider)
- [ ] Update CHANGELOG.md
- [ ] Commit, tag, push
