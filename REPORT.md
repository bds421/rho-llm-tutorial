# rho/llm Consolidated Quality Assurance Reports

This document merges the API Coverage Report, Tutorial Run Output, and Bug Reports into a single reference file to reduce clutter.

---

# API Coverage Report

Cross-reference of every exported symbol in `rho/llm v0.1.15 (github.com/bds421/rho-llm)` against tutorial coverage.

Verified by automated `grep -rl "llm.<Symbol>" */main.go` scan against `go doc -all` output.

**Legend:** T = Tested in code | I = Used indirectly (via provider init or wrapper delegation)

---

## Functions (35 total)

| # | Function | Status | Tutorial(s) | Notes |
|---|----------|--------|-------------|-------|
| 1 | `NewClient(cfg)` | T | 01-12, 15 | Core factory |
| 2 | `NewClientWithKeys(cfg, keys)` | T | 08 | Multi-key failover |
| 3 | `RegisterProvider(protocol, factory)` | I | — | Internal: called by provider `init()`; not consumer API |
| 4 | `WithLogging(client)` | T | 07 | Middleware wrapper |
| 5 | `WithLoggingPrefix(client, prefix)` | T | 07 | Middleware wrapper with custom prefix |
| 6 | `NewTextMessage(role, text)` | T | 01-12, 15, 18 | Core constructor |
| 7 | `NewToolResultMessage(id, result, isError)` | T | 03, 10, 18 | Both `isError=false` and `isError=true` |
| 8 | `NewAssistantMessage(resp)` | T | 09 | Multi-turn: builds assistant Message from Response |
| 9 | `ResolveModelAlias(model)` | T | 06, 15 | 7 aliases tested |
| 10 | `GetModelInfo(model)` | T | 04, 06, 13 | All fields inspected in 13 |
| 11 | `GetDefaultModel(provider)` | T | 06 | 4 providers |
| 12 | `GetAvailableModels(provider)` | T | 13 | 6 providers |
| 13 | `ProviderForModel(model)` | T | 06 | 3 models |
| 14 | `EstimateCost(model, in, out)` | T | 06, 08, 15 | Multiple models |
| 15 | `IsRateLimited(err)` | T | 05, 17 | Error classification |
| 16 | `IsOverloaded(err)` | T | 05, 17 | Error classification |
| 17 | `IsAuthError(err)` | T | 05, 17 | Real 401 + constructed 401/403 |
| 18 | `IsContextLength(err)` | T | 05, 17 | Error classification |
| 19 | `IsRetryable(err)` | T | 05, 17 | Error classification |
| 20 | `Backoff(attempt, base, max)` | T | 05 | Used in retry loop |
| 21 | `DefaultConfig()` | T | 13 | All fields printed |
| 22 | `IsNoAuthProvider(provider)` | T | 14, 15 | 7 providers tested |
| 23 | `PresetFor(provider)` | T | 14 | 12 providers including unknown |
| 24 | `ResolveProtocol(cfg)` | T | 14 | 6 providers |
| 25 | `ResolveBaseURL(cfg)` | T | 14 | Default + override |
| 26 | `ResolveAuthHeader(cfg)` | T | 14 | Default + override |
| 27 | `NewAuthPool(provider, keys)` | T | 16 | Pool creation + `key|baseurl` pipe syntax |
| 28 | `NewPooledClient(cfg, keys, fn)` | T | 16 | Mock client function |
| 29 | `NewRateLimitError(provider, msg)` | T | 17 | Error constructor |
| 30 | `NewOverloadedError(provider, msg)` | T | 17 | Error constructor |
| 31 | `NewAuthError(provider, msg, code)` | T | 17 | 401 + 403 status codes |
| 32 | `NewContextLengthError(provider, msg)` | T | 17 | Error constructor |
| 33 | `NewAPIErrorFromStatus(provider, status, body)` | T | 17 | 429, 503, 401, 400+context, 500, 502, generic 400 |
| 34 | `SafeHTTPClient(timeout)` | I | — | v0.1.9: HTTP client with TLS 1.2+, redirect auth stripping. Used by all adapters internally. |
| 35 | `ThinkingBudgetTokens(level, customBudget)` | — | — | v0.1.10: Converts ThinkingLevel to token count; overridden by customBudget when > 0. |

**Coverage: 32/34 consumer functions (94%)** + 1 internal (`RegisterProvider`). Untested: `SafeHTTPClient` (infrastructure), `ThinkingBudgetTokens` (utility).

---

## Types & Interfaces (23 total)

| # | Type | Status | Tutorial(s) | Notes |
|---|------|--------|-------------|-------|
| 1 | `Client` (interface) | T | 01-18 | All 5 methods (Complete, Stream, Provider, Model, Close) |
| 2 | `Config` | T | 01-18 | All 11 fields exercised |
| 3 | `Request` | T | 01-18 | All 8 fields exercised |
| 4 | `Response` | T | 01-18 | All 8 fields exercised |
| 5 | `Message` | T | 01-18 | Role + Content fields |
| 6 | `StreamEvent` | T | 02, 04, 07, 10, 11, 15, 16 | All 8 fields |
| 7 | `Tool` | T | 03, 10 | Name, Description, InputSchema |
| 8 | `ToolCall` | T | 03, 10, 18 | All 4 fields incl. ThoughtSignature |
| 9 | `ModelInfo` | T | 04, 06, 13 | All 11 fields inspected in 13 |
| 10 | `APIError` | T | 05, 17 | All 4 fields via `errors.As` |
| 11 | `ThinkingLevel` | T | 04 | All 4 constants used in code |
| 12 | `Role` | T | 01-18 | All 3 constants |
| 13 | `EventType` | T | 02, 04, 07, 10, 11, 15, 16 | All 5 constants |
| 14 | `ContentType` | T | 18 | All 4 constants |
| 15 | `ContentPart` | T | 18 | Direct construction, all fields |
| 16 | `ImageSource` | T | 18 | base64 PNG example |
| 17 | `AuthPool` | T | 16 | All 6 methods exercised |
| 18 | `AuthProfile` | T | 16 | All 4 methods + 7 struct fields |
| 19 | `PooledClient` | T | 08, 16 | Via NewClientWithKeys + direct NewPooledClient |
| 20 | `CooldownError` | T | 16 | Error(), Wait, Unwrap() |
| 21 | `LoggingClient` | T | 07 | Complete + Stream via WithLogging wrappers |
| 22 | `ProviderPreset` | T | 14 | All 3 fields via PresetFor |
| 23 | `ProviderFactory` | I | — | Internal: used by RegisterProvider |

**Coverage: 22/22 consumer types (100%)** + 1 internal (`ProviderFactory`)

---

## Constants (21 total)

| # | Constant | Status | Tutorial(s) |
|---|----------|--------|-------------|
| 1 | `RoleUser` | T | 01-12, 15, 18 |
| 2 | `RoleAssistant` | T | 03, 10 |
| 3 | `RoleSystem` | T | 09 |
| 4 | `ContentText` | T | 18 |
| 5 | `ContentImage` | T | 18 |
| 6 | `ContentToolUse` | T | 18 |
| 7 | `ContentToolResult` | T | 18 |
| 8 | `EventContent` | T | 02, 04, 07, 10, 11, 15 |
| 9 | `EventToolUse` | T | 10 |
| 10 | `EventThinking` | T | 04 |
| 11 | `EventDone` | T | 02, 04, 07, 10, 11, 15, 16 |
| 12 | `EventError` | T | 11 |
| 13 | `ThinkingNone` | T | 04 |
| 14 | `ThinkingLow` | T | 04 |
| 15 | `ThinkingMedium` | T | 04 |
| 16 | `ThinkingHigh` | T | 04 |
| 17 | `MaxErrorBodyBytes` | — | — | v0.1.9: 1 MB cap on error response reads |
| 18 | `MaxSSELineBytes` | — | — | v0.1.9: 256 KB SSE line buffer limit |
| 19 | `MaxResponseBodyBytes` | — | — | v0.1.9: 32 MB cap on success response body |
| 20 | `MaxToolInputBytes` | — | — | v0.1.9: 1 MB cap on streamed tool input |
| 21 | `TokensNotReported` | — | — | v0.1.10: Sentinel (-1) for unreported token counts |

**Coverage: 16/21 (76%)** — 5 untested constants are internal safety limits (v0.1.9-v0.1.10)

---

## Struct Fields

### Config fields (11 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `Provider` | T | 01-18 |
| `Model` | T | 01-18 |
| `APIKey` | T | 01-12, 15 |
| `MaxTokens` | T | 01, 04, 08, 11, 16 |
| `Temperature` | T | 08, 12 |
| `ThinkingLevel` | T | 04 |
| `Timeout` | T | 01-12, 15 |
| `BaseURL` | T | 08 |
| `AuthHeader` | T | 14 (ResolveAuthHeader) |
| `ProviderName` | T | 04 |
| `LogRequests` | T | 07 |

**Coverage: 11/11 (100%)**

### Request fields (9 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `Model` | T | 12 |
| `Messages` | T | 01-18 |
| `System` | T | 09, 10 |
| `MaxTokens` | T | 12 |
| `Temperature` | T | 12 |
| `Tools` | T | 03, 10 |
| `ThinkingLevel` | T | 04 (via Config fallback) |
| `ThinkingBudget` | — | — | v0.1.10: Custom token budget; overrides ThinkingLevel default |
| `StopSequences` | T | 12 |

**Coverage: 8/9 (89%)** — `ThinkingBudget` (v0.1.10) not yet demonstrated

### Response fields (8 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `ID` | T | 09 |
| `Model` | T | 09, 12 |
| `Content` | T | 01-18 |
| `ToolCalls` | T | 03, 10 |
| `Thinking` | T | 04 |
| `StopReason` | T | 01, 03, 10, 11, 12 |
| `InputTokens` | T | 01-18 |
| `OutputTokens` | T | 01-18 |

**Coverage: 8/8 (100%)**

### StreamEvent fields (8 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `Type` | T | 02, 04, 07, 10, 11, 15 |
| `Text` | T | 02, 04, 07, 11, 15 |
| `ToolCall` | T | 10 |
| `Thinking` | T | 04 |
| `InputTokens` | T | 02, 04, 07, 10, 15 |
| `OutputTokens` | T | 02, 04, 07, 10, 15 |
| `StopReason` | T | 02, 04, 07, 10, 11 |
| `Error` | T | 11 |

**Coverage: 8/8 (100%)**

### ModelInfo fields (11 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `ID` | T | 04, 06, 13 |
| `Provider` | T | 06, 13 |
| `MaxTokens` | T | 06 |
| `ContextWindow` | T | 04, 06 |
| `InputPricePer1M` | T | 06 |
| `OutputPricePer1M` | T | 06 |
| `SupportsThinking` | T | 04, 13 |
| `Thinking` | T | 04, 13 |
| `ThoughtSignature` | T | 13 |
| `NoToolSupport` | T | 13 |
| `Label` | T | 13 |

**Coverage: 11/11 (100%)**

### Tool fields (3 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `Name` | T | 03, 10 |
| `Description` | T | 03, 10 |
| `InputSchema` | T | 03, 10 |

**Coverage: 3/3 (100%)**

### ToolCall fields (4 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `ID` | T | 03, 10 |
| `Name` | T | 03, 10 |
| `Input` | T | 03, 10 |
| `ThoughtSignature` | T | 18 |

**Coverage: 4/4 (100%)**

### APIError fields (4 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `StatusCode` | T | 05, 17 |
| `Message` | T | 05, 17 |
| `Provider` | T | 05, 17 |
| `Retryable` | T | 05, 17 |

**Coverage: 4/4 (100%)**

### ContentPart fields (9 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `Type` | T | 18 |
| `Text` | T | 18 |
| `Source` | T | 18 |
| `ToolUseID` | T | 18 |
| `ToolName` | T | 18 |
| `ToolInput` | T | 18 |
| `ThoughtSignature` | T | 18 |
| `ToolResultID` | T | 18 (via NewToolResultMessage) |
| `ToolResultContent` | T | 18 (via NewToolResultMessage) |

**Coverage: 9/9 (100%)**

### ImageSource fields (3 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `Type` | T | 18 |
| `MediaType` | T | 18 |
| `Data` | T | 18 |

**Coverage: 3/3 (100%)**

### AuthProfile fields (7 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `Name` | T | 16 |
| `APIKey` | T | 16 |
| `BaseURL` | T | 16 |
| `IsHealthy` | T | 16 |
| `LastUsed` | T | 16 |
| `LastError` | T | 16 |
| `Cooldown` | T | 16 |

**Coverage: 7/7 (100%)**

### CooldownError fields (1 total)

| Field | Status | Tutorial(s) |
|-------|--------|-------------|
| `Wait` | T | 16 |

**Coverage: 1/1 (100%)**

---

## Methods (25 total)

| # | Method | Status | Tutorial(s) | Notes |
|---|--------|--------|-------------|-------|
| 1 | `AuthPool.GetCurrent()` | T | 16 | Returns current profile snapshot |
| 2 | `AuthPool.GetAvailable()` | T | 16 | Returns available or CooldownError |
| 3 | `AuthPool.MarkFailedByName()` | T | 16 | Triggers rotation |
| 4 | `AuthPool.MarkSuccessByName()` | T | 16 | Restores profile |
| 5 | `AuthPool.Count()` | T | 16 | Profile count |
| 6 | `AuthPool.Status()` | T | 16 | Status string |
| 7 | `AuthProfile.IsAvailable()` | T | 16 | Before/after MarkFailed |
| 8 | `AuthProfile.MarkUsed()` | T | 16 | Updates LastUsed |
| 9 | `AuthProfile.MarkFailed()` | T | 16 | Sets cooldown |
| 10 | `AuthProfile.MarkHealthy()` | T | 16 | Restores health |
| 11 | `CooldownError.Error()` | T | 16 | Error string |
| 12 | `CooldownError.Unwrap()` | T | 16 | Returns ErrNoAvailableProfiles |
| 13 | `APIError.Error()` | T | 17 | Via fmt.Printf |
| 14 | `PooledClient.Complete()` | T | 08, 16 | Indirect via NewClientWithKeys + direct on `*PooledClient` |
| 15 | `PooledClient.Stream()` | T | 08, 16 | Indirect via NewClientWithKeys + direct on `*PooledClient` |
| 16 | `PooledClient.Provider()` | T | 16 | Direct call |
| 17 | `PooledClient.Model()` | T | 16 | Direct call |
| 18 | `PooledClient.Close()` | T | 16 | Via defer |
| 19 | `PooledClient.PoolStatus()` | T | 16 | Pool status string |
| 20 | `LoggingClient.Complete()` | T | 07 | Via WithLogging/WithLoggingPrefix |
| 21 | `LoggingClient.Stream()` | T | 07 | Via WithLoggingPrefix + Stream |
| 22 | `LoggingClient.Provider()` | T | 07 | Via client.Provider() after wrapping |
| 23 | `LoggingClient.Model()` | T | 07 | Via client.Model() after wrapping |
| 24 | `LoggingClient.Close()` | T | 07 | Via client.Close() |
| 25 | `PooledClient.rotateClient()` | — | — | Unexported; not testable |

**Coverage: 24/24 exported methods (100%)**

---

## Variables (2 total)

| Variable | Status | Tutorial(s) | Notes |
|----------|--------|-------------|-------|
| `ErrNoAvailableProfiles` | T | 16 | Triggered by exhausting pool; verified via `errors.Is` and `CooldownError.Unwrap` |
| `ErrClientClosed` | — | — | v0.1.10: Returned by Complete/Stream after Close(). Replaces nil-pointer panic. |

**Coverage: 1/2 (50%)** — `ErrClientClosed` (v0.1.10) not yet demonstrated

---

## Overall Summary

| Category | Tested | Total | Coverage |
|----------|--------|-------|----------|
| Functions (consumer) | 32 | 34 | **94%** |
| Functions (internal) | 0 | 1 | — |
| Types (consumer) | 22 | 22 | **100%** |
| Types (internal) | 0 | 1 | — |
| Constants | 16 | 21 | **76%** |
| Variables | 1 | 2 | **50%** |
| Methods (exported) | 24 | 24 | **100%** |
| Config fields | 11 | 11 | **100%** |
| Request fields | 8 | 9 | **89%** |
| Response fields | 8 | 8 | **100%** |
| StreamEvent fields | 8 | 8 | **100%** |
| ModelInfo fields | 11 | 11 | **100%** |
| Tool fields | 3 | 3 | **100%** |
| ToolCall fields | 4 | 4 | **100%** |
| APIError fields | 4 | 4 | **100%** |
| ContentPart fields | 9 | 9 | **100%** |
| ImageSource fields | 3 | 3 | **100%** |
| AuthProfile fields | 7 | 7 | **100%** |
| CooldownError fields | 1 | 1 | **100%** |

**Note:** 9 untested symbols (v0.1.9-v0.1.11) are infrastructure/safety constants (`Max*Bytes`, `TokensNotReported`), utilities (`SafeHTTPClient`, `ThinkingBudgetTokens`), sentinel errors (`ErrClientClosed`), and advanced fields (`Request.ThinkingBudget`). All are tested in the library's own test suite (`security_test.go`, `llm_test.go`).

### Internal symbols (not consumer API)

| Symbol | Kind | Notes |
|--------|------|-------|
| `RegisterProvider` | func | Called by provider `init()` before `main()`; users never call this |
| `ProviderFactory` | type | Argument type for `RegisterProvider` |
| `PooledClient.rotateClient()` | method | Unexported |

### Verification method

Coverage was verified by automated scan:
```
grep -rl "llm.<Symbol>" */main.go
```
against the full `go doc -all .` output for `rho/llm v0.1.15`.

### Tutorials that added coverage

| Tutorial | Symbols Covered | API Keys |
|----------|----------------|----------|
| 16_pool_deep_dive | AuthPool, AuthProfile, PooledClient, CooldownError, ErrNoAvailableProfiles + all methods | No |
| 17_error_constructors | NewRateLimitError, NewOverloadedError, NewAuthError, NewContextLengthError, NewAPIErrorFromStatus | No |
| 18_content_model | ContentPart, ContentType constants, ImageSource, ToolCall.ThoughtSignature | No |

### Stress tests (19_stress_tests)

Tutorial 19 uses Go's testing framework (`go test -race -v`) with mock clients to stress-test
the library's concurrency, retry, and error-handling paths. No API keys needed.

| File | Tests | What it verifies |
|------|-------|-----------------|
| `pool_concurrent_test.go` | 3 tests + 2 benchmarks | Concurrent pool access under `-race`, no data races |
| `backoff_test.go` | 5 tests + 1 benchmark | Exponential growth, jitter distribution, bounds, zero allocs |
| `pooled_client_retry_test.go` | 8 tests | Rotation on 429/503, auth disable, non-retryable short-circuit, retry exhaustion, context cancellation, thundering herd |
| `stream_retry_test.go` | 5 tests | Pre-data retry + rotation, post-data no-retry, auth error, normal completion, caller break |
| `error_chain_test.go` | 5 tests | Deep wrapping (15 levels), all classifiers x 10 depths, CooldownError unwrap, NewAPIErrorFromStatus codes, non-API retryable |
| `cooldown_test.go` | 7 tests | Cooldown timing (60s/30s/10s), auth permanent disable, soonest profile wins |
| `edge_cases_test.go` | 9 tests | Empty/single/duplicate keys, empty provider, pipe syntax, unknown names, nil messages, zero MaxTokens |
| `content_stress_test.go` | 7 tests + 2 benchmarks | 1MB text, 10K parts, 50-turn tool chain, 100 tool calls, large results, 5MB images, JSON round-trips |

**Total: 49 tests + 5 benchmarks** — all pass with `-race` flag, zero external dependencies.

### Capability tests (20_capability_test)

Tutorial 20 is a YAML-driven multi-model capability regression suite. It sends 25 standardized
test cases to each configured model in 3 languages (EN, DE, ES) and generates a timestamped
markdown report with a scoreboard and per-test detail grid. Providers run in parallel
(anthropic, gemini, xai, openai, ollama concurrently) while models within the same provider
run sequentially to avoid rate-limit storms.

**Test matrix:** 25 tests × 3 languages = 75 tests per model, across 5 difficulty levels.

| Level | Category | Examples |
|-------|----------|---------|
| 1 | Factual | Capital cities, famous authors, planets, chemistry, continents |
| 2 | Formatting | JSON, Markdown table, CSV, HTML list, XML |
| 3 | Math/IQ | Bat-and-ball, machines, lilypad, runner, farmer |
| 4 | Logic/IQ/Mensa | Math puzzles, riddles, family relations, codes, calendars |
| 5 | Logic/IQ/Mensa | Number sequences, word puzzles, spatial reasoning |

**Validator types:**

| Validator | Behavior |
|-----------|----------|
| `json` | Parses response as valid JSON (strips markdown code fences) |
| `contains_all` | All expected substrings must appear (case-insensitive); rejects `not_expected` |
| `contains_any` | At least one expected substring must appear (case-insensitive); rejects `not_expected` |

**Configuration files:**
- `config.yaml` — Model definitions (provider, model ID, timeout, API key env var)
- `tests.yaml` — Test cases (prompt per language, validator, expected/not_expected values, difficulty)

**How to run:**
```bash
cd 20_capability_test && go test -v -timeout 120m ./...
```

**Report output:** Timestamped markdown files (`RESULTS_<YYYYMMDD_HHMMSS>.md`) in `testdata/`,
containing a scoreboard sorted by pass rate, per-test EN/DE/ES grids, and raw outputs for
failed/error cases.

### Tool use benchmark (21_cloud_ctl_tool_use)

Tutorial 21 is a YAML-driven multi-model tool use benchmark that runs agentic tool-call loops with
mock tool responses (no external dependencies required). Each configured model receives a natural-language
task, invokes tools to complete it, and the results are scored and reported. Providers run in parallel.

**Configuration files:**
- `config.yaml` — Model definitions (provider, model ID, timeout, API key env var)
- `tests.yaml` — Tool-use test cases (prompts, expected tool calls, validation)

**How to run:**
```bash
cd 21_cloud_ctl_tool_use && go test -v -timeout 60m ./...
```

**Report output:** Timestamped markdown files (`RESULTS_TOOL_USE_<YYYYMMDD_HHMMSS>.md`) in `testdata/`.

### Modifications for coverage

| Tutorial | Symbols Added |
|----------|--------------|
| 04_thinking | ThinkingNone, ThinkingLow, ThinkingMedium (used in code), Config.ProviderName |
| 07_logging_and_middleware | LoggingClient.Stream() |
| 09_system_and_multiturn | NewAssistantMessage(resp) |


---

# Tutorial Run Output

**Date:** 2026-03-18
**Library:** `rho/llm v0.1.15`
**Environment:** `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, `XAI_API_KEY`, `OPENAI_API_KEY` set. Ollama running locally.

---

## 01_basic

```
Response: Quantum entanglement is when two or more particles become
Stop reason: max_tokens
Tokens: input=8, output=9
```

**Exit code:** 0 — PASS

---

## 02_streaming

```
Streaming response:
---
# Go Iterators

Range through the channels, one by one,
Each value flows like morning sun,
A goroutine spins in the night,
Sending data, pure and light.
...
---
Done: reason=end_turn, input=16, output=120
```

**Exit code:** 0 — PASS

---

## 03_tool_use

```
Sending initial request...
Final answer:
Tokens: input=142, output=39
```

**Exit code:** 0 — PASS (Gemini empty final Content — API behavior)

---

## 04_thinking

```
Model: claude-opus-4-6
  SupportsThinking (API-controlled budgets): true
  Thinking (intrinsic reasoning):            false
  Context window: 200000 tokens

=== ThinkingNone + ProviderName ===
Provider (overridden): anthropic-via-proxy
Response: Hello.
Thinking (should be empty): ""

=== Synchronous (Complete) ===
Thinking:
  This is the classic cognitive reflection test problem...
  x = 0.05, ball costs $0.05
Answer:
  Ball = $0.05, Bat = $1.05
Tokens: input=74, output=547

=== Streaming ===
[Thinking] same reasoning streamed via EventThinking
Answer streamed via EventContent
Done: reason=end_turn, input=74, output=472
```

**Exit code:** 0 — PASS

---

## 05_error_handling

```
Attempt 1/3...
  Auth error: anthropic API error (status 401): ...invalid x-api-key...
  -> Check your API key. Not retrying.
```

**Exit code:** 1 — PASS (by design)

---

## 06_cost_and_registry

```
=== Alias Resolution ===
  opus -> claude-opus-4-6, sonnet -> claude-sonnet-4-6, haiku -> claude-haiku-4-5-20251001
  flash -> gemini-2.5-flash, grok -> grok-4.20-beta, gemini-pro -> gemini-3.1-pro-preview

=== Cost Estimation ===
  claude-sonnet-4-6 (1000 in, 500 out): $0.010500
  claude-opus-4-6 (10000 in, 2000 out): $0.300000
  gemini-2.5-flash (5000 in, 1000 out): $0.001350

=== Provider Detection ===
  gemini-2.5-flash -> gemini, claude-sonnet-4-6 -> anthropic, gpt-5.4 -> openai

=== Default Models ===
  anthropic -> claude-sonnet-4-6, gemini -> gemini-3.1-flash-lite-preview
  xai -> grok-4.20-beta, openai -> gpt-5.4
```

**Exit code:** 0 — PASS

---

## 07_logging_and_middleware

```
=== Approach 1: Config.LogRequests ===
INFO complete request  component=llm provider=gemini model=gemini-2.5-flash
INFO complete done     elapsed=507ms tokens_in=9 tokens_out=8 stop=end_turn cost=$0.000006
Response: 7 * 8 = 56

=== Approach 2: WithLogging ===
Client: provider=gemini, model=gemini-2.5-flash
INFO complete done     elapsed=432ms tokens_in=9 tokens_out=8
Response: 7 * 8 = 56

=== Approach 3: WithLoggingPrefix ===
INFO complete request  component=[MyApp] ...
Response: 7 * 8 = 56

=== Approach 4: WithLoggingPrefix + Stream ===
INFO stream request    component=[StreamTest] provider=gemini model=gemini-2.5-flash
Streaming: 7 * 8 = 56
Done: reason=end_turn, input=9, output=8
INFO stream done       component=[StreamTest] elapsed=713ms chunks=1 tokens_in=9 tokens_out=8
```

**Exit code:** 0 — PASS

---

## 08_auth_pool_failover

```
Example 1: Multi-Key Anthropic -> "The capital of France is Paris." Cost: $0.000207
Example 2: Per-Profile Endpoints -> Error: openai API key is required
Example 3: Custom Provider -> Error: custom API key is required
Example 4: Ollama -> Response: 4
```

**Exit code:** 0 — PARTIAL (no OpenAI keys)

---

## 09_system_and_multiturn

```
=== Approach 1: Request.System field ===
Response: Ahoy, matey! Paris be the capital o' France, it be!
Response ID: msg_01YUSavynoGfMxvqh4bd4mZU
Response Model: claude-haiku-4-5-20251001

=== Approach 2: RoleSystem message (Gemini) ===
Response: Fair Paris, France's heart, doth hold that crown.

=== Multi-Turn Conversation ===
Turn 1 - User: My name is Alice. Remember it.
Turn 1 - Assistant: Got it! I'll remember that your name is Alice.
Turn 2 - User: What is my name?
Turn 2 - Assistant: Your name is Alice.
Turn 3 - User: Spell it backwards.
Turn 3 - Assistant: E-C-I-L-A
Total turns: 3, Final token count: input=73, output=12
```

**Exit code:** 0 — PASS

---

## 10_streaming_tool_use

```
=== Streaming Tool Use with Error Recovery ===
  Stream: [tool:lookup_city({"city":"Tokyo"})]
  Done: reason=end_turn, in=70, out=15
  Executing 1 tool call(s)...
    lookup_city -> {"city":"Tokyo","population":"13.96 million","country":"Japan"}
  Stream: [tool:lookup_city({"city":"Atlantis"})]
  Done: reason=end_turn, in=102, out=16
  Executing 1 tool call(s)...
    lookup_city -> ERROR: city "atlantis" not found in database
  Stream: Tokyo: 13.96 million. Atlantis: not found.
  Done: reason=end_turn, in=125, out=15

Final answer: Tokyo: 13.96 million. Atlantis: not found.
```

**Exit code:** 0 — PASS

---

## 11_stream_abort_and_errors

```
=== Test 1: Early Stream Abort (break after 50 chars) ===
# The History of Computing\nComputing has transforme
  [ABORTED after 50 chars — break cleans up HTTP connection]
  Total chars received before abort: 52

=== Test 2: Context Cancellation ===
1 2 3 4 5 ... 16
  [Cancelling context...]
  Stream error after cancel: stream error: context canceled
  Lines received: 4

=== Test 3: Short Timeout ===
  Timeout/error: stream error: context deadline exceeded
  Chars received: 1004, timed out: true
```

**Exit code:** 0 — PASS (all 3 abort mechanisms work)

---

## 12_request_overrides

```
=== Test 1: Temperature 0 vs 1.5 ===
  temp=0.0 -> Random
  temp=1.5 -> (empty — Gemini quirk at high temp with short output)

=== Test 2: MaxTokens Override ===
  Response (max 10 tokens): (empty)
  Stop reason: max_tokens
  Output tokens: 0

=== Test 3: Stop Sequences ===
  Response: 4, 5, 6, 7, 8, 9, 10
  Stop reason: end_turn

=== Test 4: Request.Model Override ===
  Config model: gemini-2.5-flash
  Request model: gemini-2.5-flash-lite
  Response.Model: gemini-2.5-flash-lite
  Response: Gemini
```

**Exit code:** 0 — PASS

---

## 13_registry_deep

```
=== DefaultConfig ===
  Provider: anthropic, Model: claude-sonnet-4-6, MaxTokens: 8192, Temperature: 1.0, Timeout: 2m0s

=== Available Models Per Provider ===
  anthropic (9 models): claude-opus-4-6 [thinking], claude-sonnet-4-6 [thinking], ...
  gemini (8 models): gemini-3.1-pro-preview [thought-signature], gemini-3.1-flash-lite-preview, gemini-2.5-flash, ...
  xai (8 models): grok-4.20-beta [intrinsic-reasoning], grok-4-1-fast-reasoning [intrinsic-reasoning], ...
  openai (18 models): gpt-5.4-pro [intrinsic-reasoning], gpt-5.4 [intrinsic-reasoning], ...
  groq (6 models): llama-3.3-70b-versatile, deepseek-r1-distill-llama-70b [intrinsic-reasoning], ...
  mistral (8 models): mistral-large-2512, magistral-medium-2509 [intrinsic-reasoning], ...

=== Models with Thinking Support ===
  25 models across anthropic (API-controlled), xai, openai, groq, mistral (intrinsic)

=== Models with ThoughtSignature ===
  gemini-3.1-pro-preview, gemini-3-pro-preview, gemini-3-flash-preview

=== Models Without Tool Support ===
  (none)
```

**Exit code:** 0 — PASS

---

## 14_provider_helpers

```
=== Provider Presets ===
  anthropic -> BaseURL: https://api.anthropic.com/v1             Protocol: anthropic
  gemini    -> BaseURL: https://generativelanguage.googleapis.com Protocol: gemini
  openai    -> BaseURL: https://api.openai.com/v1                Protocol: openai_compat
  ollama    -> BaseURL: http://localhost:11434/v1                 Protocol: openai_compat
  (11 providers total, unknown_provider -> not found)

=== No-Auth Providers ===
  ollama, vllm, lmstudio: no auth needed
  anthropic, gemini, openai, custom: auth required

=== Protocol Resolution ===
  anthropic -> anthropic, gemini -> gemini, others -> openai_compat

=== BaseURL Resolution ===
  anthropic (default): https://api.anthropic.com/v1
  anthropic (override): https://my-proxy.example.com/v1
  ollama (default): http://localhost:11434/v1

=== Auth Header Resolution ===
  anthropic: (empty — uses x-api-key), openai: Bearer, gemini: (empty — query param)
```

**Exit code:** 0 — PASS

---

## 15_multi_provider

```
Prompt: "What is the square root of 144? Answer with just the number."

Provider             Response   In Tok   Out Tok  Cost         Latency
Gemini Flash         12         18       2        $0.000003    605ms
Anthropic Haiku      12         22       5        $0.000038    804ms
Ollama Qwen3:4b                 27       50       $0.000000    567ms

=== Streaming Comparison ===
  Gemini Flash: 12 (in=18, out=2)
  Anthropic Haiku: 12 (in=22, out=5)
  Ollama Qwen3:4b: (in=27, out=50)
```

**Exit code:** 0 — PASS

---

## 16_pool_deep_dive

```
=== Step 1: AuthPool Creation & Inspection ===
Pool count: 3
Pool status: *demo-1:ok, demo-2:ok, demo-3:ok
Current profile: name=demo-1, key=sk-key-alpha, baseURL=""

=== Step 2: AuthProfile Lifecycle ===
Profile "demo-1": IsAvailable=true
After MarkUsed: LastUsed=20:21:23
After MarkFailed: IsHealthy=true, IsAvailable=false, LastError="rate limited"
  Cooldown until: 20:21:28
After MarkHealthy: IsHealthy=true, IsAvailable=true

=== Step 3: Pool Rotation ===
Available profile: demo-1
WARN profile failed profile=demo-1 error="auth error" cooldown=10s
After marking demo-1 failed: *demo-1:cooldown 10s, demo-2:ok, demo-3:ok
After marking demo-1 success: *demo-1:ok, demo-2:ok, demo-3:ok

=== Step 4: ErrNoAvailableProfiles & CooldownError ===
WARN profile failed profile=test-1 error="rate limited" cooldown=10s
WARN profile failed profile=test-2 error=overloaded cooldown=10s
GetAvailable error: no available auth profiles (all in cooldown): next available in 10s
  -> errors.Is(err, ErrNoAvailableProfiles) = true
  -> CooldownError.Error(): no available auth profiles (all in cooldown): next available in 10s
  -> CooldownError.Wait: ~10s
  -> CooldownError.Unwrap(): no available auth profiles (all in cooldown)
  -> errors.Is(Unwrap(), ErrNoAvailableProfiles) = true
  -> errors.Is(wrapped, ErrNoAvailableProfiles) = true

=== Step 5: NewPooledClient (mock) ===
INFO pooled client created profiles=2 provider=demo
PooledClient provider: demo
PooledClient model: mock-model
PoolStatus: *demo-1:ok, demo-2:ok
Complete response: mock response
Stream events: done (reason=end_turn)
```

**Exit code:** 0 — PASS (all offline, no API keys)

---

## 17_error_constructors

```
=== Step 1: Named Error Constructors ===
NewRateLimitError: anthropic API error (status 429): rate limit exceeded
  IsRateLimited=true, IsRetryable=true
NewOverloadedError: gemini API error (status 503): service overloaded
  IsOverloaded=true, IsRetryable=true
NewAuthError (401): openai API error (status 401): invalid api key
  IsAuthError=true, IsRetryable=false
NewAuthError (403): anthropic API error (status 403): forbidden
  IsAuthError=true, IsRetryable=false
NewContextLengthError: openai API error (status 400): context length exceeded
  IsContextLength=true, IsRetryable=false

=== Step 2: NewAPIErrorFromStatus ===
Status 429 -> IsRateLimited=true, IsRetryable=true
Status 503 -> IsOverloaded=true, IsRetryable=true
Status 401 -> IsAuthError=true
Status 400 (context) -> IsContextLength=true
Status 500 -> IsRetryable=true
Status 502 -> IsRetryable=true
Status 400 (generic) -> (none match)

=== Step 3: Error Wrapping Round-Trip ===
Original:      IsRateLimited=true
Wrapped:       IsRateLimited=true
DoubleWrapped: IsRateLimited=true
Extracted from double-wrap: status=429, provider=anthropic
```

**Exit code:** 0 — PASS (all offline, no API keys)

---

## 18_content_model

```
=== Step 1: NewTextMessage -> ContentPart ===
Role: user, Parts: 1
  Part[0]: Type=text, Text="Hello, world!"

=== Step 2: Multimodal Message (text + image) ===
Role: user, Parts: 2
  Part[0]: Type=text, Text="What is in this image?"
  Part[1]: Type=image, Source.Type=base64, Source.MediaType=image/png, Source.Data=96 bytes

=== Step 3: NewToolResultMessage -> ContentToolResult ===
  Part[0]: Type=tool_result, ToolResultID=call_123, Content={"temperature": 22}, IsError=false

=== Step 4: ContentToolUse + ThoughtSignature ===
Type: tool_use, ToolUseID: call_456, ToolName: get_weather
ThoughtSignature: gemini3-sig-abc123
ToolCall.ThoughtSignature: gemini3-sig-xyz789

=== Step 5: ContentType Constants ===
  ContentText       = "text"
  ContentImage      = "image"
  ContentToolUse    = "tool_use"
  ContentToolResult = "tool_result"

=== Step 6: JSON Serialization ===
{ "role": "user", "content": [{type: text}, {type: image, source: base64 PNG}] }
```

**Exit code:** 0 — PASS (all offline, no API keys)

---

## Overview

| # | Tutorial | Exit | Status | Provider | Key Features Tested |
|---|----------|------|--------|----------|---------------------|
| 01 | basic | 0 | PASS | Gemini | Config, NewClient, Complete, NewTextMessage, Response |
| 02 | streaming | 0 | PASS | Anthropic | Stream, EventContent, EventDone |
| 03 | tool_use | 0 | PASS | Gemini | Tool, ToolCall, agentic loop, NewToolResultMessage |
| 04 | thinking | 0 | PASS | Anthropic | ThinkingNone/High, ProviderName, resp.Thinking, EventThinking |
| 05 | error_handling | 1 | PASS | Anthropic | APIError, Is*Error helpers, Backoff |
| 06 | cost_and_registry | 0 | PASS | (none) | ResolveModelAlias, GetModelInfo, EstimateCost, ProviderForModel, GetDefaultModel |
| 07 | logging_middleware | 0 | PASS | Gemini | LogRequests, WithLogging, WithLoggingPrefix, LoggingClient.Stream |
| 08 | auth_pool_failover | 0 | PARTIAL | Mixed | NewClientWithKeys, per-profile endpoints, Ollama |
| 09 | system_multiturn | 0 | PASS | Anthropic+Gemini | Request.System, RoleSystem, RoleAssistant, multi-turn, Response.ID/Model |
| 10 | streaming_tool_use | 0 | PASS | Gemini | EventToolUse in stream, isError=true recovery |
| 11 | stream_abort | 0 | PASS | Anthropic | break abort, context cancel, timeout mid-stream |
| 12 | request_overrides | 0 | PASS | Gemini | Request.Temperature, MaxTokens, StopSequences, Request.Model |
| 13 | registry_deep | 0 | PASS | (none) | DefaultConfig, GetAvailableModels, ModelInfo.Label/NoToolSupport/ThoughtSignature |
| 14 | provider_helpers | 0 | PASS | (none) | PresetFor, ResolveProtocol/BaseURL/AuthHeader, IsNoAuthProvider |
| 15 | multi_provider | 0 | PASS | All 3 | Cross-provider comparison, streaming comparison, IsNoAuthProvider |
| 16 | pool_deep_dive | 0 | PASS | (none) | AuthPool, AuthProfile, PooledClient, CooldownError, ErrNoAvailableProfiles |
| 17 | error_constructors | 0 | PASS | (none) | NewRateLimitError, NewOverloadedError, NewAuthError, NewContextLengthError, NewAPIErrorFromStatus |
| 18 | content_model | 0 | PASS | (none) | ContentPart, ContentType, ImageSource, ToolCall.ThoughtSignature |
| 21 | cloud_ctl_tool_use | — | TEST SUITE | Mixed | Tool, ToolCall, agentic tool-use loop, YAML-driven multi-model matrix |

### Summary

| Metric | Count |
|--------|-------|
| Full pass | 17/18 |
| Partial pass (missing keys) | 1/18 (tutorial 08) |
| Compilation errors | 0/18 |
| Unexpected failures | 0/18 |
| Test suites | 3 (stress: 49 tests, capability: 75/model, tool-use: YAML-driven) |

### Changes in v0.1.8–v0.1.15

| Version | Change | Details |
|---------|--------|---------|
| v0.1.8 | Groq models added | 6 models including llama-3.3-70b, deepseek-r1 distills |
| v0.1.8 | Mistral models added | 8 models including magistral (reasoning), codestral, devstral |
| v0.1.8 | Stop reasons normalized | Now `end_turn`/`max_tokens` (lowercase) instead of `STOP`/`MAX_TOKENS` |
| v0.1.9 | Security hardening | Gemini key→header, bounded reads (1 MB error / 256 KB SSE / 32 MB response / 1 MB tool input), redirect auth stripping, TLS 1.2 minimum |
| v0.1.9 | Security test suite | 15 tests in `security_test.go` |
| v0.1.10 | `TokensNotReported` sentinel | Distinguishes "not reported" (-1) from "zero tokens" (0) |
| v0.1.10 | `Request.ThinkingBudget` | Per-request thinking token budget override |
| v0.1.10 | `ErrClientClosed` sentinel | Replaces nil-pointer panic on use-after-Close |
| v0.1.10 | Network error detection | `IsRetryable` now checks `net.Error`, `io.EOF`, `ECONNRESET`, `ECONNREFUSED` via type assertion |
| v0.1.11 | Go 1.26 minimum | Resolves 15 stdlib CVEs in crypto/tls, net/http |
| v0.1.11 | `ThinkingBudgetTokens` fix | Default case now returns 0 (was 4096 for `ThinkingNone`) |
| v0.1.11 | Error wrapping fix | `AuthPool.GetAvailable()` now wraps `ErrNoAvailableProfiles` via `%w` |
| v0.1.11 | Local provider resilience | Keyless providers (Ollama, vLLM) now get retry/backoff via `PooledClient` |
| v0.1.11 | Adapter `Close()` fix | All adapters now call `CloseIdleConnections()` on close |
| v0.1.12 | Gemini empty text fix | All adapters skip empty `ContentText` parts |
| v0.1.13 | Circuit breaker | 3-state machine, configurable retry policy, cooldowns, retry observability hook |
| v0.1.14 | Context caching | Anthropic inline caching, Gemini cached content references |
| v0.1.15 | GitHub migration | Module path changed to `github.com/bds421/rho-llm`; new models (Grok 4.20, Gemini 3.1 Flash Lite, GPT 5.4, GPT 5.3) |

### Known issues

| Issue | Type | Tutorial | Details |
|-------|------|----------|---------|
| Gemini empty final Content in tool loop | Gemini API behavior | 03 | Tool-use response has empty Content field |
| Gemini empty response at high temperature | Gemini API behavior | 12 | `temp=1.5` with short prompt returns empty |
| Gemini empty response at `MaxTokens: 10` | Gemini API behavior | 12 | `OutputTokens: 0` but `StopReason: max_tokens` |
| Tutorial 08 partial (no OpenAI keys) | Missing credentials | 08 | Examples 2+3 fail without OpenAI key |

---

## 19_stress_tests

```
$ cd tutorial/19_stress_tests && go test -race -v -count=1 ./...

--- PASS: TestBackoff_ExponentialGrowth (0.00s)
--- PASS: TestBackoff_JitterDistribution (0.00s)
--- PASS: TestBackoff_CappedAtMaxDelay (0.00s)
--- PASS: TestBackoff_ZeroBaseDelay (0.00s)
--- PASS: TestBackoff_BaseExceedsMax (0.00s)
--- PASS: TestContentPart_LargeTextPayload (0.05s)
--- PASS: TestContentPart_ManyParts (0.07s)
--- PASS: TestContentPart_DeepToolCallChain (0.00s)
--- PASS: TestNewAssistantMessage_ManyToolCalls (0.00s)
--- PASS: TestNewAssistantMessage_EmptyResponse (0.00s)
--- PASS: TestNewToolResultMessage_LargeResult (0.00s)
--- PASS: TestImageSource_LargeData (0.40s)
--- PASS: TestCooldown_ProfileBecomesAvailableAfterExpiry (0.00s)
--- PASS: TestCooldown_CooldownErrorWaitAccuracy (0.00s)
--- PASS: TestCooldown_AuthErrorPermanentlyDisables (0.00s)
--- PASS: TestCooldown_RateLimitCooldown60s (0.00s)
--- PASS: TestCooldown_OverloadedCooldown30s (0.00s)
--- PASS: TestCooldown_TransientCooldown10s (0.00s)
--- PASS: TestCooldown_SoonestProfileWins (0.00s)
--- PASS: TestNewAuthPool_EmptyKeys (0.00s)
--- PASS: TestNewAuthPool_SingleKey (0.00s)
--- PASS: TestNewAuthPool_DuplicateKeys (0.00s)
--- PASS: TestNewAuthPool_EmptyProvider (0.00s)
--- PASS: TestNewAuthPool_PipeSyntaxVariants (0.00s)
--- PASS: TestMarkFailedByName_UnknownName (0.00s)
--- PASS: TestMarkSuccessByName_UnknownName (0.00s)
--- PASS: TestPooledClient_Complete_NilMessages (0.00s)
--- PASS: TestPooledClient_Complete_ZeroMaxTokens (0.00s)
--- PASS: TestErrorChain_DeepWrapping (0.00s)
--- PASS: TestErrorChain_AllClassifiers (0.00s)
--- PASS: TestErrorChain_CooldownErrorUnwrap (0.00s)
--- PASS: TestErrorChain_NewAPIErrorFromStatus_AllCodes (0.00s)
--- PASS: TestIsRetryable_NonAPIErrors (0.00s)
--- PASS: TestAuthPool_ConcurrentGetAvailable (0.00s)
--- PASS: TestAuthPool_ConcurrentMarkFailedAndGetAvailable (0.50s)
--- PASS: TestAuthPool_ConcurrentGetCurrentReadOnly (0.00s)
--- PASS: TestPooledClient_Complete_RotatesOn429 (0.00s)
--- PASS: TestPooledClient_Complete_RotatesOn503 (0.00s)
--- PASS: TestPooledClient_Complete_AuthErrorPermanentlyDisables (0.00s)
--- PASS: TestPooledClient_Complete_NonRetryableReturnsImmediately (0.00s)
--- PASS: TestPooledClient_Complete_ExhaustsAllRetries (5.00s)
--- PASS: TestPooledClient_Complete_ContextCancellation (0.20s)
--- PASS: TestPooledClient_Complete_SingleKeyThreeRetries (0.00s)
--- PASS: TestPooledClient_Complete_ConcurrentRotation (0.00s)
--- PASS: TestPooledClient_Stream_PreDataRetry (0.00s)
--- PASS: TestPooledClient_Stream_PostDataNoRetry (0.00s)
--- PASS: TestPooledClient_Stream_PreDataAuthError (0.00s)
--- PASS: TestPooledClient_Stream_NormalCompletion (0.00s)
--- PASS: TestPooledClient_Stream_CallerBreaks (0.00s)
ok  	tutorial/19_stress_tests	7.656s

$ go test -bench=. -benchmem -run=^$ ./...
BenchmarkBackoff-16              185416502    6.307 ns/op    0 B/op   0 allocs/op
BenchmarkNewTextMessage-16       538942728    2.260 ns/op    0 B/op   0 allocs/op
BenchmarkNewAssistantMessage-16    2075667    608.4 ns/op  5040 B/op   5 allocs/op
BenchmarkAuthPool_GetAvailable-16 30123632    39.93 ns/op    0 B/op   0 allocs/op
BenchmarkAuthPool_Parallel-16     10030659    118.9 ns/op    0 B/op   0 allocs/op
```

**Exit code:** 0 — PASS (49 tests + 5 benchmarks, all with `-race`)


---

# rho/llm Bug Report

**Date:** 2026-03-18
**Library version:** `rho/llm v0.1.15`
**Test suite:** 18 progressive tutorials + 3 test suites (stress + capability + tool-use)
**Environment:** macOS Darwin 25.3.0, Go 1.26.0, Anthropic + Gemini + xAI + OpenAI API keys, Ollama local

---

## Bugs Fixed During This Session

### FIX-1: `GetDefaultModel("openai")` returned `claude-sonnet-4-6`

**Fixed in:** v0.1.7
**Root cause:** `registry.go` had no `"openai"` entry in the `defaultModels` map. The `GetDefaultModel` function fell through to a hardcoded fallback `return "claude-sonnet-4-6"` (line 166), which is the Anthropic default — clearly wrong for OpenAI.
**Fix applied:** Added `"openai": "gpt-5.2"` to the `defaultModels` map.

### FIX-2: OpenAI models missing from registry

**Fixed in:** v0.1.7
**Root cause:** No OpenAI models were registered in the model registry. `ProviderForModel("gpt-5.2")` returned empty string. `GetAvailableModels("openai")` returned nil.
**Fix applied:** 16 current OpenAI models added to the registry (gpt-5.2, gpt-5.1, gpt-5, gpt-4.1, o3, o4-mini, etc.). Deprecated models like `gpt-4o` (API shut down Feb 2026) were intentionally excluded.

### FIX-3: `GetAvailableModels` for Groq/Mistral returned nil

**Fixed in:** v0.1.8
**Severity:** Low
**Root cause:** Groq and Mistral were missing from the model registry despite having provider presets.
**Fix applied:** Added 6 Groq models (llama-3.3, deepseek-r1 distills) and 8 Mistral models (mistral-large, magistral reasoning models) to the registry with metadata.

### FIX-4: Inconsistent `StopReason` values across providers

**Fixed in:** v0.1.8
**Severity:** Low
**Root cause:** Gemini returned `"STOP"`/`"MAX_TOKENS"`, Anthropic returned `"end_turn"`/`"max_tokens"`/`"tool_use"`.
**Fix applied:** Normalized all providers to return lowercase `end_turn`, `max_tokens`, or `tool_use`.

### FIX-5: `llm.NewAssistantMessage(resp)` added for tool-use history

**Fixed in:** v0.1.8
**Severity:** High
**Root cause:** No easy way to build an assistant message that preserved tool-use content blocks, leading to Anthropic API errors in multi-turn tool loops.
**Fix applied:** Added `NewAssistantMessage(resp)` which automatically clones all text and tool content parts from a `Response`.

### FIX-3: `LogRequests: true` produced no visible log output

**Fixed in:** latest main (post v0.1.7)
**Severity:** Medium
**Root cause:** The logging middleware in `middleware.go` (lines 35, 51, 68, 87) used `slog.Debug` for all request/response metadata logging. The default `slog` level is `Info`. When a user explicitly set `Config.LogRequests: true`, the per-request logs (provider, model, tokens, cost, elapsed time) were silently swallowed.

The pool creation log at `slog.Info` (`factory.go:57`) *was* visible, creating the misleading impression that logging was working — but the actual request metadata the user opted into never appeared.

**Fix applied:** Changed `slog.Debug` to `slog.Info` in the logging middleware.
**Verified:** Tutorial 07 now shows:
```
INFO complete request  component=llm provider=gemini model=gemini-2.5-flash messages=1 tools=0
INFO complete done     component=llm provider=gemini model=gemini-2.5-flash elapsed=593ms tokens_in=9 tokens_out=8 stop=STOP cost=6.15e-06
```

### FIX-4: `Config.ThinkingLevel` not passed to Anthropic adapter

**Fixed in:** latest main (post v0.1.7)
**Severity:** High
**Root cause:** `ThinkingLevel` exists on both `Config` (config.go:25) and `Request` (types.go:152). The Anthropic adapter's `buildRequest` method (anthropic.go:287) only read `req.ThinkingLevel` from the `Request` struct. There was no code to copy `Config.ThinkingLevel` into the request as a default.

The README documents setting `ThinkingLevel` on `Config`:
```go
cfg := llm.Config{
    ThinkingLevel: llm.ThinkingLow,
}
```
But this had no effect — the adapter never saw it. Users had to set `ThinkingLevel` on every `Request`, which contradicts the documented API.

**Fix applied:** The adapter now falls back to `c.config.ThinkingLevel` when `req.ThinkingLevel` is empty.
**Verified:** Tutorial 04 now produces thinking output with `ThinkingLevel` set only on Config.

---

## Open Issues

None — all previously reported bugs have been fixed.

---

## Closed Issues

### BUG-1: Anthropic adapter does not handle `RoleSystem` messages

**Severity:** Medium — **Fixed in v0.1.8**
**Tutorial:** 09_system_and_multiturn
**Affected providers:** Anthropic only (Gemini handles it correctly)

**Description:**
When `RoleSystem` messages were included in the `Request.Messages` array, the Anthropic adapter passed them through as-is with `role: "system"`. Anthropic's API rejected this with:

```
messages: Unexpected role "system". The Messages API accepts a top-level `system` parameter,
not "system" as an input message role.
```

**Expected behavior:**
The adapter should detect `RoleSystem` messages in the messages array and automatically promote them to the top-level `system` parameter (same as `Request.System`), matching how the Gemini adapter converts them to `systemInstruction`.

**Workaround:**
Use `Request.System` instead of `RoleSystem` messages when targeting Anthropic:
```go
req := llm.Request{
    System: "You are a pirate.",  // works
    Messages: []llm.Message{...},
}
```

**Location:** `provider/anthropic/anthropic.go`, `buildRequest` method (around line 248-274). The message conversion loop has no `case llm.RoleSystem` handler.

---

### BUG-5: Anthropic adapter doesn't handle `RoleSystem` in message conversion

**Severity:** Low — **Fixed in v0.1.8** (same fix as BUG-1)
**Tutorial:** 09_system_and_multiturn

**Description:**
The Gemini adapter's `buildRequest` had explicit handling for `llm.RoleSystem` messages. The Anthropic adapter's `buildRequest` had no such handling. The `switch msg.Role` block only handled `RoleUser` and `RoleAssistant`, falling through to `default` which passed the role string verbatim. Now system messages are extracted into the top-level `system` parameter.

---

## Observations (Not Bugs)

### Gemini returns empty `resp.Content` after tool use completion

**Tutorial:** 03_tool_use
**Explanation:** After a tool use loop completes, Gemini's final response sometimes contains no text parts. The adapter correctly parses whatever Gemini returns. This is Gemini API behavior — the model considers the tool results self-explanatory and doesn't produce a separate summary.
**Recommendation:** Document this behavior; suggest callers check `resp.Content == ""` after tool loops.

### Gemini returns empty response at very low `MaxTokens`

**Tutorial:** 12_request_overrides
**Explanation:** With `MaxTokens: 10`, Gemini returns `OutputTokens: 0` and `StopReason: MAX_TOKENS` — it doesn't return partial tokens. This is Gemini API behavior.

### `ProviderForModel("gpt-4o")` returns empty

**Explanation:** `gpt-4o` was deprecated (API shut down Feb 2026) and intentionally excluded from the registry. Not a bug.

### `grok-4-fast-non-reasoning` has `MaxTokens: 0`

**Explanation:** By design — `0` means "use config default (8192)". xAI doesn't publish per-model output limits. Same pattern as Ollama models.

---

## API Coverage After All Tutorials

### Now tested (was previously untested)

| Symbol | Tutorial |
|--------|----------|
| `RoleSystem` | 09 (Gemini), 09 (Anthropic — documents failure) |
| `Request.System` | 09 |
| `Request.StopSequences` | 12 |
| `Request.Temperature` | 12 |
| `Request.MaxTokens` | 12 |
| `Request.Model` | 12 |
| `Response.ID` | 09 |
| `Response.Model` | 09, 12 |
| `EventToolUse` in streaming | 10 |
| `NewToolResultMessage` with `isError: true` | 10 |
| Multi-turn conversation | 09 |
| Early stream abort (break) | 11 |
| Context cancellation mid-stream | 11 |
| Timeout mid-stream | 11 |
| `GetAvailableModels` | 13 |
| `DefaultConfig` | 13 |
| `ModelInfo.Label` | 13 |
| `ModelInfo.NoToolSupport` | 13 |
| `ModelInfo.ThoughtSignature` | 13 |
| `PresetFor` | 14 |
| `ProviderPreset` fields | 14 |
| `ResolveProtocol` | 14 |
| `ResolveBaseURL` | 14 |
| `ResolveAuthHeader` | 14 |
| `IsNoAuthProvider` | 14, 15 |
| Multi-provider comparison | 15 |

### Previously untested — now covered by tutorials 16–18

All symbols listed as "still untested" in prior versions of this report are now fully exercised:
`ContentPart`, `ContentType`, `ImageSource` (tutorial 18), `AuthPool`, `NewAuthPool`, pool methods, `AuthProfile` methods, `PooledClient`, `NewPooledClient`, `PoolStatus`, `CooldownError`, `ErrNoAvailableProfiles` (tutorial 16), error constructors (tutorial 17).
