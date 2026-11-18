# Tutorial 26: PDF input across providers

Sends one generated PDF ("The secret word is PELICAN.") to every provider whose
API key is set and prints the answer plus the wire part each host received.

Demonstrates `llm.ContentDocument`, `DocumentSource`, `ContentPart.Title`,
`llm.ValidateDocumentSource` and `llm.RequireCapabilities(...CapabilityDocumentInput)`.

## Run

```sh
source ../.env                 # ANTHROPIC / GEMINI / OPENAI / MISTRAL / XAI keys; missing ones are skipped
go run .                       # the demo
go test -short -race ./...     # offline: httptest servers capture the exact request bodies
go test -run Live -v ./...     # live: one cheap call per provider (needs keys)
```

## How each host receives the PDF

| Provider / model | Wire part | Title becomes |
|---|---|---|
| anthropic `claude-haiku-5-5` | `{"type":"document","source":{"type":"base64",...}}` | `title` |
| gemini `gemini-3.8-flash` | `{"inlineData":{"mimeType":"application/pdf","data":...}}` | not sent |
| openai `gpt-6-luna` (Responses) | `{"type":"input_file","filename","file_data"}` | `filename` |
| openai `gpt-4.1` (Chat Completions) | `{"type":"file","file":{"filename","file_data"}}` | `filename` (default `document-N.pdf`) |
| mistral `mistral-medium-latest` | `{"type":"document_url","document_url":<data URL>}` | not sent |
| xai `grok-4.7` | refused before any request (Chat Completions takes no files) | n/a |

OpenAI-shaped `file` is also used for OpenRouter and unknown hosts. Mistral is
detected by provider name or by a `BaseURL` containing `api.mistral.ai`; xAI by
`xai`/`grok` or `api.x.ai`. Before rho-llm v0.20.0 every OpenAI-compatible host got
the PDF as an `image_url` data URI and rejected it (xAI 400, Mistral 422, OpenAI 400).

For xAI, extract the text and send it as a text part, or send page images.

## Things to know (pinned by the tests)

- `ValidateDocumentSource` checks media type, canonical base64 and the `%PDF-`
  signature only; a body that starts with `%PDF-` passes even if it is not a real PDF.
- Text documents (`llm.NewTextDocument`) are Anthropic-only; every other adapter refuses them.
- Models unknown to the registry (custom provider names, OpenRouter slugs) need
  `Config.ModelCapabilities` before rho-llm will dispatch to them.
- `Title` is sent verbatim as the filename; sanitise untrusted titles yourself.
- A PDF in an assistant message is rejected by Anthropic, silently dropped by
  openai_compat and Responses (unvalidated), and forwarded by Gemini.
