# Tutorial 27: Anthropic Message Threads

`llm.WithMessageThreads()` makes a `Session` use the Anthropic beta
`message-threads-2026-08-12`. The API keeps the conversation for about five
minutes, so a turn sends only what is new:

| Turn | Request |
|------|---------|
| 1 | full request, `thread: {type: "create"}` |
| 2+ | only the new messages, `thread: {type: "continue", previous_message_id: <last reply ID>}`; `system` and `tools` are left off the wire |

The Session still holds the whole transcript. It continues only when the
transcript prefix, system prompt, tools, model and generation settings match
what the server holds and under 4m30s have passed; otherwise it sends a full
request as a new thread. A continuation answered with `thread_not_found`,
`thread_already_continued`, `thread_continue_in_flight` or any other 400/404 is
replayed once in full; `thread_unsupported_request` turns threads off for the
Session. Refusals, fallback-model replies and replies without an ID end the
thread; `SwitchProvider` drops it. Off by default, Anthropic only (inert on
other providers), `Conversation` JSON is unchanged.

Direct clients use `Request.Thread` (`&llm.Thread{}` creates,
`&llm.Thread{PreviousMessageID: id}` continues), and `llm.ThreadErrorCode(err)`
returns the `Thread*` code of a 400/404.

## What it saves

Request size, not billed tokens: the server rebuilds the same prompt, so input
tokens are identical with and without threads (live: 5067 vs 5067 over 4
turns), while request bodies shrink (12349 vs 3703 bytes) because the history
and the system/tools are not re-uploaded.

## Run

```bash
source ../.env && go run .          # live demo (ANTHROPIC_API_KEY)
go test -short -race ./...          # offline, hostile tests against an httptest fake
go test -run Live -v ./...          # live test (needs the key; skipped under -short)
```

Without the key `go run .` explains the feature and exits 0.

## Tests

`fake_test.go` is an `httptest` fake of the Messages API with thread
semantics (each ID continues once, no system/tools on a continue, unknown ID is
404) that records every request body and `anthropic-beta` header and mirrors
the thread the server holds, so tests assert the server's copy equals the
Session's transcript. Covered: delta-only continue and ID chaining, every
fallback, unsupported-off, 4xx/5xx that are not thread fallbacks, pool retry
keeping the continuation, changed system prompt / model / resumed conversation,
Send and Stream, tool loops, SwitchProvider, non-Anthropic hosts, identical
Conversation JSON, missing ID / refusal / truncated JSON / cut stream / garbage
bodies, and concurrent Sends under `-race`.

Not testable from outside: the 4m30s expiry uses an unexported Session clock
and is covered only by the library's own `TestThreadsExpireByTheClock`.
