## Model Names

Always use full model IDs (e.g. `"gemini-2.5-flash"`, `"claude-haiku-4-5-20251001"`) in tutorial code.
Never use short aliases (`"flash"`, `"haiku"`, `"sonnet"`, etc.) — except in tutorial 06 which
demonstrates `ResolveModelAlias`.

## Environment & API Keys

- A local `.env` file in the project root contains all API keys (Anthropic, Gemini, OpenAI, xAI, Mistral).
- Always `source .env` before running any tutorial or test that hits a cloud provider.
- Ollama runs locally and needs no API key.

## Testing

- **Start small**: test one tutorial with one local model (ollama) + one cloud model first, then expand.
- **Short tests**: use `go test -short ./...` — the full capability/IQ test (`20_capability_test`) is very long-running.
- **Each tutorial is its own Go module** with its own `go.mod` — update deps per-module.
- **19_stress_tests** uses mocks and is safe to run without API keys or `-short`.
- **20_capability_test** is the full IQ test — never run it casually; it's expensive and slow.

## Git Workflow

- **Always `git pull` first** before starting any work — ensure you have the latest remote changes.

## Releases

- **Always update `CHANGELOG.md`** before committing a version bump or tagged release.
- Follow the existing format: `## vX.Y.Z — YYYY-MM-DD` with `### Added`, `### Changed`, `### Fixed` sections.
