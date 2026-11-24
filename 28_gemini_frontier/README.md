# 28 — Gemini frontier: Gemini 4, long decode, the Interactions API

Covers what rho-llm v0.19.0 and v0.20.0 added for Google's newest surface.

| Topic | Behaviour |
|---|---|
| Models | `gemini-4-carbon`, `gemini-4-hydrogen` (1,048,576-token window; `ModelInfo.MaxTokens` is the *per-request* output limit) |
| Thinking levels | `xhigh` exists on Hydrogen and Neon (`ThinkingXHigh`/`ThinkingMax` send `high` elsewhere). `minimal` is a 400 on 3.8 Flash, 3.7 Flash and every Gemini 4 model, so it is sent as `low`. |
| Caching | Gemini 4 has no explicit caching: `Request.CachedContent` is refused before the request. |
| Long decode | A Gemini 4 hop can end with `finishReason: CONTINUATION` + `continuationToken`. The adapter resends the same request with the token (top level, contents unchanged), joins text and tool calls, sums usage, stops at 64 hops (reported as `max_tokens`). Works for `Complete` and `Stream`. |
| Finish reasons | `ESCALATION`, `PUP_LIMITED_DISABLED` -> `refusal`; `MISSING_THOUGHT_SIGNATURE`, `MALFORMED_RESPONSE` -> `error`; `RawStopReason` keeps the original. |
| Sampling | `SamplingParams` keys `temperature`, `top_p`, `top_k`, `topP`, `topK` are refused on every Gemini model (Google answers 400 "Unknown name": they are merged at the top level, Gemini wants them in `generationConfig`). Use `Request.Temperature` (refused up front on models without sampling controls). |
| Interactions API | Opt-in provider `gemini-interactions` (protocol `gemini_interactions`, `POST /v1beta/interactions`). `gemini`/`google` stay on generateContent. `store:false` by default; `SamplingParams["store"] = true` opts in. Structured output, `CachedContent`, `ThinkingBudget`, `Effort` are refused. |

## Run

```bash
cd 28_gemini_frontier
go run .                      # steps 1-3 offline; steps 4-5 need GEMINI_API_KEY
go test -short -race ./...    # offline: httptest fakes, no key, no network
set -a; . ../.env; set +a
go test -count=1 -run Live -v ./...   # a handful of tiny live calls
```

Long decode needs a server that answers `CONTINUATION`, so it lives in the tests
(`longdecode_test.go`): multi-hop `Complete`/`Stream`, token without
CONTINUATION, CONTINUATION without token, endless continuation (hop cap), a later
hop failing / malformed / cut mid-stream, consumer break, negative or missing
usage, retries, and concurrency under `-race`.

Two library bugs found by these tests were fixed in rho-llm v0.20.0 and stay as
regression tests: an empty later hop discarded the earlier hops' text, and the
interactions provider did not refuse top-level sampling keys.
