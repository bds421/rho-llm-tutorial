# Tutorial 24 — Speech-to-Text Compare

A small local web app that sends one recording to several speech-to-text models
in parallel through rho-llm's provider-neutral `ModalityClient.TranscribeAudio`,
and shows the results side by side.

| Column | Provider / model | Vocabulary goes to |
|---|---|---|
| Google | `gemini-3.5-transcribe` (dedicated STT) | `customVocabulary` |
| Google | `gemini-3.5-flash-lite` (multimodal chat model asked to transcribe) | the instruction |
| OpenAI | `gpt-transcribe`, `gpt-4o-mini-transcribe` | Whisper-style `prompt` |
| xAI | `grok-voice-transcribe-2.0` (`POST /v1/stt`) | `keyterm` fields |
| Anthropic | — | The Claude API has no audio input or STT endpoint |

## Run

Keys come from the repository root `.env` (`GEMINI_API_KEY`, `OPENAI_API_KEY`,
`XAI_API_KEY`); a column without its key is shown as not configured.

```bash
cd 24_speech_to_text
go run . -addr 127.0.0.1:8090   # then open http://127.0.0.1:8090
```

Press **Record**, speak, press **Stop** — the clip is transcribed automatically.
**Transcribe again** re-sends the same clip (for example after toggling the
vocabulary). Enter a reference text to get each model's word error rate; words
not in the reference are highlighted and the best column is outlined.

Every request is logged to `stt.log` (git-ignored): one line per provider with
latency and transcript or error, plus a request line with status and size.

## What each request costs

Every Stop/Transcribe sends the clip to **every** configured model and spends
API credit on each. The cost column uses approximate list prices per audio
minute (October 2026), for comparison only: Gemini 3.5 Transcribe ≈ $0.005,
Gemini 3.5 Flash Lite ≈ $0.0006 (audio tokens), gpt-transcribe ≈ $0.0045,
gpt-4o-mini-transcribe ≈ $0.003, Grok ≈ $0.0017. Check the providers' pricing
pages before relying on them.

## Safety

- Binds to loopback only; any other `-addr` is refused (the page has no auth
  and spends credit).
- The media type comes from the uploaded bytes (`llm.AudioMediaTypeFromSignature`),
  not the browser header; unknown containers, oversize uploads (14 MiB, also for
  chunked bodies) and malformed language tags are rejected before any paid call.
- Closing the tab cancels the in-flight provider calls.

## Language

Gemini receives the full BCP 47 tag (`de-AT`). OpenAI and xAI take ISO-639-1
codes, so they receive the primary subtag (`de`); rho-llm deliberately rejects a
region tag for them rather than altering it silently.

## Findings from a first test (German, one speaker, 3 clips)

With a vocabulary list, all models were close; no model was error-free in every
clip. Grok was the fastest (~0.8 s), Gemini 3.5 Transcribe ~1.9 s. The most
damaging error was a wrong quantity ("150 g" for "50 g") under fast speech, and
a vocabulary entry can over-correct ("Huhn" → "Hühnerfleisch"). Keep the list to
words the speaker actually uses, and let users review the transcript.

## Tests

```bash
make test-stt   # or: go test -race ./...
```

Offline break tests with fake providers: invalid uploads and languages never
reach a provider, chunked oversize bodies are capped, a failing or slow
provider does not affect the other columns, client disconnects cancel calls,
concurrent requests are race-free, and only loopback addresses are accepted.
