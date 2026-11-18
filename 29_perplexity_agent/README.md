# Tutorial 29: Perplexity Presets (Sonar and the Agent API)

Two Perplexity presets in rho-llm v0.20.0:

| Preset | Endpoint | Protocol | Models |
|---|---|---|---|
| `perplexity` | `https://api.perplexity.ai/chat/completions` | `openai_compat` | `sonar`, `sonar-pro`, `sonar-reasoning-pro`, `sonar-deep-research` |
| `perplexity-agent` (opt-in) | `https://api.perplexity.ai/v1/responses` | `openai_responses` + Perplexity dialect | vendor-prefixed IDs: `perplexity/sonar`, `perplexity/kimi-k3`, `anthropic/claude-sonnet-5-5`, `google/gemini-4-*` ... |

The dialect sends the system prompt as `instructions`, keeps `max` reasoning effort,
adds `/v1` to a bare-host `BaseURL`, and prices `input_tokens_details.cached_tokens`
as cache reads. `effort` / `max_steps` ride `Request.SamplingParams`. A model from one
catalog is refused on the other preset before any HTTP request.

## Run

```
go run .                       # offline demo; real calls only with PERPLEXITY_API_KEY
go test -short -race ./...     # offline, hostile httptest fakes
go test -run TestLive ./...    # live; Perplexity tests skip without PERPLEXITY_API_KEY,
                               # the OpenAI stream check needs OPENAI_API_KEY
```

## Honesty note

The Agent API wire shapes (paths, `instructions`, event names, `cached_tokens`) come
from Perplexity's documentation as encoded in rho-llm. **They were not verified against
the live Perplexity API** by this tutorial; the Perplexity live tests exist but were
never run (no key). The offline tests prove rho-llm's behaviour against fakes of those
documented shapes, not Perplexity's.

## Findings (v0.20.0)

- `SamplingParams["tools"]` is refused (reserved key), so Perplexity built-in Agent tools
  cannot be passed that way yet (`TestAgentToolsViaSamplingParamsIsRefused`).
- A `BaseURL` ending in `/v1/` used to produce `/v1//responses`; fixed in rho-llm v0.20.0
  (`TestAgentBaseURLTrailingSlashV1`).
- The pooled client ends a stream at its first error, including a malformed SSE line.
