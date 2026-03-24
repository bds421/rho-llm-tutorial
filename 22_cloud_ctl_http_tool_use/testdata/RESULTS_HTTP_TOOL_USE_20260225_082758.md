# Cloud-CTL HTTP Tool Use Benchmark Report

Generated: `2026-02-25 08:27:58`

## Tool Definitions

| Tool | HTTP Endpoint | Description |
|---|---|---|
| `drive_ls` | `GET /api/v1/files` | List files and folders in cloud storage. Maps to `GET /ap... |
| `drive_search` | `GET /api/v1/files?q=Q` | Search for files in cloud storage using Google Drive API ... |
| `email_search` | `GET /api/v1/emails?q=Q` | Search emails by query. Maps to `GET /api/v1/emails?q=Q`.... |
| `calendar_list` | `GET /api/v1/calendar/events` | List calendar events in a date range. Maps to `GET /api/v... |
| `calendar_calendars` | `GET /api/v1/calendar/calendars` | List all available calendars. Maps to `GET /api/v1/calend... |
| `sheets_read` | `GET /api/v1/sheets/:id` | Read data from a spreadsheet. Maps to `GET /api/v1/sheets... |

## Test Scenarios

| ID | Level | Category | Prompt | Expected Tools |
|---|---|---|---|---|
| `single-list-files` | 1 | Single Tool | List my recent files. | drive_ls |
| `single-search-email` | 1 | Single Tool | Show my unread emails. | email_search |
| `single-list-calendars` | 1 | Single Tool | What calendars do I have? | calendar_calendars |
| `args-search-pdf` | 2 | Arguments | Find all PDF files in my drive. | drive_search |
| `args-email-from-subject` | 2 | Arguments | Find emails about the budget. | email_search |
| `args-calendar-date-range` | 2 | Arguments | Show my calendar events for this week. | calendar_list |
| `args-drive-search-keyword` | 2 | Arguments | Search for spreadsheets in my drive. | drive_search |
| `multi-meetings-emails` | 3 | Multi-Tool | What meetings do I have this week, and are ther... | calendar_list, email_search |
| `multi-files-events` | 3 | Multi-Tool | List my recent files and also show this week's ... | drive_ls, calendar_list |
| `multi-calendars-events` | 3 | Multi-Tool | Show me all my calendars, then list this week's... | calendar_calendars, calendar_list |
| `error-invalid-sheet` | 4 | Error Recovery | Read the spreadsheet with ID invalid-xyz-999. | sheets_read |
| `error-empty-search` | 4 | Error Recovery | Search my drive for files named 'xylophonenebul... | drive_search |
| `complex-weekly-summary` | 5 | Complex Reasoning | Summarize my week: what meetings do I have, any... | calendar_list, email_search, drive_ls |
| `complex-project-overview` | 5 | Complex Reasoning | Give me a project overview: check for recent em... | email_search, calendar_list, drive_ls |
| `complex-day-prep` | 5 | Complex Reasoning | Help me prepare for today: what's on my calenda... | calendar_list, email_search, drive_ls |

## Scoreboard

| Model | Provider | Pass Rate | Tool Acc | Arg Acc | Avg Rounds | Avg Latency | Pass | Fail | Err |
|---|---|---|---|---|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | anthropic | 100% | 100% | 100% | 2.1 | 7334ms | 15 | 0 | 0 |
| `claude-opus-4-6` | anthropic | 100% | 100% | 100% | 2.5 | 19458ms | 15 | 0 | 0 |
| `claude-sonnet-4-6` | anthropic | 100% | 100% | 100% | 2.3 | 14005ms | 15 | 0 | 0 |
| `grok-3-mini` | xai | 100% | 100% | 100% | 1.3 | 22978ms | 15 | 0 | 0 |
| `grok-4-1-fast-non-reasoning` | xai | 100% | 100% | 100% | 1.3 | 5925ms | 15 | 0 | 0 |
| `qwen3:8b` | ollama | 73% | 100% | 100% | 2.3 | 40316ms | 11 | 4 | 0 |
| `gemini-3-flash-preview` | gemini | 0% | 0% | 80% | 0.0 | 2246ms | 0 | 15 | 0 |
| `gemini-3.1-pro-preview` | gemini | 0% | 0% | 80% | 0.0 | 6724ms | 0 | 15 | 0 |

## Detailed Results

### single-list-files (Level 1 - Single Tool)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | drive_ls | 1 | 6466ms |
| `claude-opus-4-6` | PASS | drive_ls | 1 | 7782ms |
| `claude-sonnet-4-6` | PASS | drive_ls | 1 | 6331ms |
| `grok-3-mini` | PASS | drive_search | 1 | 17301ms |
| `grok-4-1-fast-non-reasoning` | PASS | drive_ls | 1 | 5105ms |
| `qwen3:8b` | PASS | drive_ls | 1 | 34988ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 1839ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 7126ms |

### single-search-email (Level 1 - Single Tool)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | email_search | 1 | 4576ms |
| `claude-opus-4-6` | PASS | email_search | 1 | 9521ms |
| `claude-sonnet-4-6` | PASS | email_search | 1 | 8615ms |
| `grok-3-mini` | PASS | email_search | 1 | 12514ms |
| `grok-4-1-fast-non-reasoning` | PASS | email_search | 1 | 3600ms |
| `qwen3:8b` | PASS | email_search | 1 | 19806ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 1907ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 4385ms |

### single-list-calendars (Level 1 - Single Tool)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | calendar_calendars | 1 | 3264ms |
| `claude-opus-4-6` | PASS | calendar_calendars | 1 | 9349ms |
| `claude-sonnet-4-6` | PASS | calendar_calendars | 1 | 7010ms |
| `grok-3-mini` | PASS | calendar_calendars | 1 | 15332ms |
| `grok-4-1-fast-non-reasoning` | PASS | calendar_calendars | 1 | 4454ms |
| `qwen3:8b` | PASS | calendar_calendars | 1 | 11582ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 1178ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 3245ms |

### args-search-pdf (Level 2 - Arguments)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | drive_search, drive_ls | 2 | 5932ms |
| `claude-opus-4-6` | PASS | drive_search | 1 | 7186ms |
| `claude-sonnet-4-6` | PASS | drive_search | 1 | 4405ms |
| `grok-3-mini` | PASS | drive_search | 2 | 20946ms |
| `grok-4-1-fast-non-reasoning` | PASS | drive_search | 1 | 2735ms |
| `qwen3:8b` | PASS | drive_search | 1 | 8637ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 1456ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 4522ms |

### args-email-from-subject (Level 2 - Arguments)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | email_search | 1 | 5376ms |
| `claude-opus-4-6` | PASS | email_search | 1 | 13100ms |
| `claude-sonnet-4-6` | PASS | email_search | 1 | 10876ms |
| `grok-3-mini` | PASS | email_search | 1 | 21738ms |
| `grok-4-1-fast-non-reasoning` | PASS | email_search | 1 | 4408ms |
| `qwen3:8b` | PASS | email_search | 1 | 38558ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 1192ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 3775ms |

### args-calendar-date-range (Level 2 - Arguments)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | calendar_list, calendar_calendars | 4 | 6908ms |
| `claude-opus-4-6` | PASS | calendar_list | 3 | 13716ms |
| `claude-sonnet-4-6` | PASS | calendar_list, calendar_calendars | 5 | 17006ms |
| `grok-3-mini` | PASS | calendar_list, calendar_calendars | 3 | 40550ms |
| `grok-4-1-fast-non-reasoning` | PASS | calendar_list | 2 | 6103ms |
| `qwen3:8b` | PASS | calendar_list, calendar_calendars | 10 | 154866ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 2076ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 6326ms |

### args-drive-search-keyword (Level 2 - Arguments)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | drive_search | 1 | 7418ms |
| `claude-opus-4-6` | PASS | drive_search | 1 | 8235ms |
| `claude-sonnet-4-6` | PASS | drive_search | 1 | 8165ms |
| `grok-3-mini` | PASS | drive_search | 1 | 17101ms |
| `grok-4-1-fast-non-reasoning` | PASS | drive_search | 1 | 5192ms |
| `qwen3:8b` | PASS | drive_search | 1 | 8339ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 1530ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 9855ms |

### multi-meetings-emails (Level 3 - Multi-Tool)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | calendar_list, email_search, calendar_calendars | 3 | 8562ms |
| `claude-opus-4-6` | PASS | calendar_list, email_search, calendar_calendars | 6 | 41986ms |
| `claude-sonnet-4-6` | PASS | calendar_list, email_search, calendar_calendars | 4 | 23005ms |
| `grok-3-mini` | PASS | calendar_list, email_search | 1 | 24348ms |
| `grok-4-1-fast-non-reasoning` | PASS | calendar_list, email_search, calendar_calendars | 3 | 11303ms |
| `qwen3:8b` | PASS | calendar_list, email_search | 1 | 25114ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 2876ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 8224ms |

### multi-files-events (Level 3 - Multi-Tool)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | drive_ls, calendar_list | 2 | 8919ms |
| `claude-opus-4-6` | PASS | drive_ls, calendar_list, calendar_calendars | 4 | 24565ms |
| `claude-sonnet-4-6` | PASS | drive_ls, calendar_list | 2 | 12051ms |
| `grok-3-mini` | PASS | drive_search, calendar_list | 2 | 53941ms |
| `grok-4-1-fast-non-reasoning` | PASS | drive_ls, calendar_list | 1 | 7857ms |
| `qwen3:8b` | PASS | drive_search, calendar_list, drive_ls | 4 | 120247ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 2427ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 7826ms |

### multi-calendars-events (Level 3 - Multi-Tool)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | calendar_calendars, calendar_list | 2 | 5516ms |
| `claude-opus-4-6` | PASS | calendar_calendars, calendar_list | 3 | 17401ms |
| `claude-sonnet-4-6` | PASS | calendar_calendars, calendar_list | 2 | 11953ms |
| `grok-3-mini` | PASS | calendar_calendars, calendar_list | 1 | 31610ms |
| `grok-4-1-fast-non-reasoning` | PASS | calendar_calendars, calendar_list | 1 | 5609ms |
| `qwen3:8b` | PASS | calendar_calendars, calendar_list | 8 | 113734ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 3099ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 8784ms |

### error-invalid-sheet (Level 4 - Error Recovery)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | sheets_read | 1 | 2597ms |
| `claude-opus-4-6` | PASS | sheets_read | 1 | 7033ms |
| `claude-sonnet-4-6` | PASS | sheets_read | 1 | 5905ms |
| `grok-3-mini` | PASS | sheets_read | 1 | 9261ms |
| `grok-4-1-fast-non-reasoning` | PASS | sheets_read | 1 | 2564ms |
| `qwen3:8b` | PASS | sheets_read | 1 | 8050ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 1579ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 4273ms |

### error-empty-search (Level 4 - Error Recovery)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | drive_search | 1 | 2485ms |
| `claude-opus-4-6` | PASS | drive_search | 1 | 6210ms |
| `claude-sonnet-4-6` | PASS | drive_search | 1 | 4159ms |
| `grok-3-mini` | PASS | drive_search | 1 | 14610ms |
| `grok-4-1-fast-non-reasoning` | PASS | drive_search | 1 | 1886ms |
| `qwen3:8b` | FAIL | drive_search | 1 | 6209ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 1946ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 5757ms |

### complex-weekly-summary (Level 5 - Complex Reasoning)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | calendar_list, email_search, drive_ls, calendar_calendars | 4 | 16791ms |
| `claude-opus-4-6` | PASS | calendar_list, email_search, drive_ls, calendar_calendars | 4 | 41780ms |
| `claude-sonnet-4-6` | PASS | calendar_list, email_search, drive_ls, calendar_calendars | 4 | 30096ms |
| `grok-3-mini` | PASS | calendar_list, email_search, drive_search | 1 | 20612ms |
| `grok-4-1-fast-non-reasoning` | PASS | calendar_list, email_search, drive_search, drive_ls | 2 | 8953ms |
| `qwen3:8b` | FAIL | calendar_list, email_search, drive_search | 1 | 14588ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 4644ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 7981ms |

### complex-project-overview (Level 5 - Complex Reasoning)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | email_search, calendar_list, drive_ls, calendar_calendars | 4 | 12954ms |
| `claude-opus-4-6` | PASS | email_search, calendar_list, drive_ls, calendar_calendars | 6 | 47454ms |
| `claude-sonnet-4-6` | PASS | email_search, calendar_list, drive_ls, calendar_calendars | 5 | 29789ms |
| `grok-3-mini` | PASS | email_search, calendar_list, drive_search | 1 | 22372ms |
| `grok-4-1-fast-non-reasoning` | PASS | email_search, calendar_list, drive_ls | 1 | 8817ms |
| `qwen3:8b` | FAIL | email_search, calendar_list, drive_search | 1 | 19054ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 2259ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 9030ms |

### complex-day-prep (Level 5 - Complex Reasoning)

| Model | Status | Tools Called | Rounds | Latency |
|---|---|---|---|---|
| `claude-haiku-4-5-20251001` | PASS | calendar_list, email_search, drive_ls, calendar_calendars | 3 | 12241ms |
| `claude-opus-4-6` | PASS | calendar_list, email_search, drive_ls, calendar_calendars | 4 | 36549ms |
| `claude-sonnet-4-6` | PASS | calendar_list, email_search, drive_ls, calendar_calendars | 5 | 30705ms |
| `grok-3-mini` | PASS | calendar_list, email_search, drive_search | 1 | 22432ms |
| `grok-4-1-fast-non-reasoning` | PASS | calendar_list, email_search, drive_search, drive_ls | 2 | 10290ms |
| `qwen3:8b` | FAIL | calendar_list, email_search, drive_search | 1 | 20972ms |
| `gemini-3-flash-preview` | FAIL |  | 0 | 3686ms |
| `gemini-3.1-pro-preview` | FAIL |  | 0 | 9758ms |

## Raw Tool Call Traces (Failed/Error Tests)

<details><summary>Click to expand</summary>

#### `gemini-3-flash-preview` -- single-list-files [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- single-search-email [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- single-list-calendars [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- args-search-pdf [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- args-email-from-subject [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- args-calendar-date-range [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- args-drive-search-keyword [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- multi-meetings-emails [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- multi-files-events [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- multi-calendars-events [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- error-invalid-sheet [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- error-empty-search [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- complex-weekly-summary [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- complex-project-overview [Fail]

**Final Response:**
```

```

#### `gemini-3-flash-preview` -- complex-day-prep [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- single-list-files [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- single-search-email [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- single-list-calendars [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- args-search-pdf [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- args-email-from-subject [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- args-calendar-date-range [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- args-drive-search-keyword [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- multi-meetings-emails [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- multi-files-events [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- multi-calendars-events [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- error-invalid-sheet [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- error-empty-search [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- complex-weekly-summary [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- complex-project-overview [Fail]

**Final Response:**
```

```

#### `gemini-3.1-pro-preview` -- complex-day-prep [Fail]

**Final Response:**
```

```

#### `qwen3:8b` -- error-empty-search [Fail]

**Tool Calls:**
```json
[
  {
    "name": "drive_search",
    "input": {
      "query": "name contains 'xylophonenebulatractor'"
    },
    "output": "null",
    "is_error": false
  }
]
```

**Final Response:**
```
Your cloud drive does not contain any files named "xylophonenebulatractor". The search did not return any matching results. Double-check the filename for typos or try a different search term if needed.
```

#### `qwen3:8b` -- complex-weekly-summary [Fail]

**Tool Calls:**
```json
[
  {
    "name": "calendar_list",
    "input": {
      "from": "2026-02-22",
      "to": "2026-02-28"
    },
    "output": "{\"error\":\"bad request: Bad Request\"}",
    "is_error": true
  },
  {
    "name": "email_search",
    "input": {
      "query": "is:unread"
    },
    "output": "[{\"id\":\"19c9376fead35da9\",\"thread_id\":\"19c9376fead35da9\",\"subject\":\"Der Frühling kommt mit neuen Terminen! \",\"from\":\"EICHENWERK \\u003coffice@eichenwerk.at\\u003e\",\"to\":[\"\\u003creneheinzl@gmail.com\\u003e\"],\"date\":\"2026-02-25T07:22:54+01:00\",\"body\":\"Runter von der Couch \\u0026 ab zur Range - Das heißt es mit dem kommenden Frühling und unseren neuen Terminen!  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­\\r\\n\\r\\nMit dem Frühling kommen auch die neuen Termine!\\r\\n\\r\\nRunter von der Couch \\u0026 ab zur Range! Das heißt es mit dem nahenden Frühling - der nicht nur gutes Wetter mit sich bringt, sondern auch unsere neuen Termine.\\r\\n\\r\\nHESA VERTEIDIGUNGSSCHIESSEN\\r\\n\\r\\nDas Basisseminar und Sondermodule.\\r\\n\\r\\nModul 1 bis 4  (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3m3W2RkTxx3JdwWzMjx7rP5xrLcVcVF7b2jcHYmW2Jk4y38mQ59BW9bQDNy5xFc9pW4G5Szn5JNgK4W2tc8bh8b3hgCW7tN_GG7GDZ5GW3w-9W33zvG3sW4S9tDW2h3twqW5JFjhW64J2y8W2V4p-Z83zBF3W2TFVJv5SgLJlW1snT-42W7jb6W1ZNQ8Y5PW0nKW713HzV1yg0mKN1lv7gK65VdrW7VhJq94d6Dz0N3NLrVXrNSbPW5mSGWQ6XJMCbVGSJgQ2lLPcVW849lN63D1QxFW4LbFpt3C_kfRW8pvfDd6jSKmvf1PSQD604 ) – sicheres Schießen von Anfang an, bis hin zum Schießen in der Bewegung und in Stresssituationen.\\r\\n\\r\\nModul 6 (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10Y3lcq-W95jsWP6lZ3mVW4FYyqF6xw9tPVnXfbt1vdM5pW21j2SS8fSG5CW44T1Lw6Lb39nW5KwsNJ7VsvY6W3kzyCG46BpQSW5fv4Kc7q8N4vW31fCCn5HdQgxVpQhdH7sM-Z6W3kswPl7qwb_nW6Q7Rl-7QHvJCW44zGfk1VsyL8W2yf3Kg7RFhwJVdylt35QqKnpW2D-2Nr3pTXsKW3TxLR35Hjc79W4-LHTF1QdfqtW4whrhL1DbL-1V-nnYc7Hq6BMW69Ghy06KS66FW3qY_q43q_d-kW27Vg3X1Xpx-tW5zHZ3_3fybjSVz9Wwn4zhv7CW8hzRkv46MGjHW249k5B28GRdhW6z0-ts2QvDSFW3-GmGD7VQks2W1zg5lP135_nyW6kKl6P8ssBtZf1kdCJF04 ) – Notwehrsituationen im Eigenheim: Szenarientraining, Videoanalyse und rechtliche Aufklärung.\\r\\n\\r\\nGrundschulung Gewehr  (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10Y3lcq-W95jsWP6lZ3pWW23JhYZ6pX-jTW5LQZx082mtPrMQSxR47XZz0VpgFY97lPJc5W10mzry3SMdTkW6T5WF94YGPlhW8h0S1g940WrxF6nMjn-kB9mW2DLNZs9dzKGWW6C7CYr2YvVTTW92LYj420wdtHW5rWtbq3JNV7pVLk-Dj7SkfNdN87YR8spsCtpW65PRK36rHFQqW7XjC4g800c_0VsYzj8586fMtW8BytpS6jd3RlW2cHR4R3yl-sgW95cCNc1wPHFnVf1gFg1n5-lmN985h_T8sRKmW6R8wSL9lLJMQW60xbBF4lkktPW4MBDfC4VYS4jN3NldCPr6jy7W5rb9Vk2tKFDvW8pr-4J8hrNVxW5dtTL47Q36hsW8T0HvG6GBzb_f28VMNl04 ) Langwaffenausbildungen für Anfänger (Grundschulung) und Fortgeschrittene (Modul 1-2-3)\\r\\n\\r\\nZU DEN KURSEN\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3pCW5W0TG_9dWky4N4Pn7KV5-FHTW2YHM5r2nn2YsVxjbgx5Jh4Y8W4wMjJx3fXXSGW2Dl5DK4sd3czW1zvJdh88TFh_W7Vktk74gDttJW38mSQ17Bh9trW8YRHV92KNNwyW3T7wrX98TyNzW4_TMCP3G-qkQW4yWy8Y4dp330W1KGZNW78fkdKW1McBLv55t0NNW6XfQQY6d2kZTW5rdsyY2vYcg5Vp0xxw8mwm7FW5dXvzW41CSsPN6frdYSDQPqlW1jhy0p71GBcFVLtV4j4fW4dbW1K_Rn_5n7VzJW11XVn63VXDWqf77VRFR04 )\\r\\n\\r\\nModul 9 Pistole.\\r\\n\\r\\nConcealed carry (verdeckte Trageweise) wir zeigen dir im Kurs unterschiedliche Präsentationstechniken, verschiedene Holster und trainieren direkt diverse Drills.\\r\\n\\r\\nMEHR ERFAHREN\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10F3lcq-W8wLKSR6lZ3kqW2GN4D73mW2GdW6KQTs64L7bHYW1-W8Sj7sWvJbW1GbHrH6rSgQMW8N8TWN6Rw3GSW5BFG-L8vt1MsW1D3wlC3f2WKZN1C64BFPQQFYW5Ksjn31M9S-nW20G-ts27L94wW17Ynpt6mnrxgW86HxXn2G4b0WVWVfWM6bMqXkM2-Jlvy6hbSV9xHtk73w6T-W1J-XvV3P4YJPW3T6M6v7c1FzqW7-VS7Q12g64nW2hPjSz3RHWRNW4h1mZq7Krr6MW2WYcP21BC085W38tk0Y3BZqfdW82LNMN89VydlW8zL0Rb6vpgFzV7VGjJ35QmFlW7Pkpx27ptWJgW1NgkLF83FMZZMh2rvbMCcVYf62j-fl04 )\\r\\n\\r\\nTraining mit Köpfchen.\\r\\n\\r\\nZum Fördern deiner Schießfertigkeit, Wahrnehmungs- und Konzentrationsfähigkeit!\\r\\n\\r\\nMEHR ERFAHREB\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10F3lcq-W8wLKSR6lZ3lWW3WWcP66fbBVNW6LQ--B854DbxW6d6xRs7cBfvFW8tP5ny88-dxtW7kkjN-20RgBBW4vQNx51Jz5KDW13y8kY7zpg6FVlNjR07DDRtCV6pmkB3KbdTPW4SK5X861SnK6W8L9VKp3KBtxhW2w1hVD4LqcQnW7NbjF069RyzgW45Ht2c4vpgLbN2t4tn5Fv6QMW3KxN2y67rdf7W91b2wp2k3tVbW6mMmMm60y5zxVkXL_R5hbWjZW1YBYYF1r14QqMx8Nd71vXz2W1ykJQq17lT74W86QJb34pBv0GVTfYws7tHNbgW2zMDl72yj-q1V5Ztmx9k0MpBN1vnjxLJ_qnnW5402mq2FDrzzf7R37Bl04 )\\r\\n\\r\\n1.png (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3m6VrB0Y12mn9NxW4fqL_Q4Tf0_kN3hjVbS-KyZZW307Gqk8ptZRhW671khG79bbkyW5n54Kt628q4lW8WsKlM26wlcJW1Fc-QX3kTq0MVZmWQQ4wG5z4W5X2-HZ5hLHGrW5XhNYq2B_xwwW99QqHC4gsbkvN27FYxz8V1KfW5PB_Gv3mg2yWW17bzz586vC8MW5KfvC03VWRG2W42yqDV72F0sYW7T7CyQ618Kt2W2mnKt58xpM92W6wdtQY8ykDh_W5NxBMR882F3lW83mhdT8jVN83MwydZZDT_X9W81pmJ72zV0jRf8nFFLg04 )\\r\\n\\r\\nSurvivalrally 2026\\r\\n\\r\\nSichere dir dein Ticket für die Survivalrally 2026 und werde Teil des größten Bushcraft \\u0026 Survival Events in Österreich.\\r\\n\\r\\nMEHR ERFAHREN\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3m6VrB0Y12mn9NxW4fqL_Q4Tf0_kN3hjVbS-KyZZW307Gqk8ptZRhW671khG79bbkyW5n54Kt628q4lW8WsKlM26wlcJW1Fc-QX3kTq0MVZmWQQ4wG5z4W5X2-HZ5hLHGrW5XhNYq2B_xwwW99QqHC4gsbkvN27FYxz8V1KfW5PB_Gv3mg2yWW17bzz586vC8MW5KfvC03VWRG2W42yqDV72F0sYW7T7CyQ618Kt2W2mnKt58xpM92W6wdtQY8ykDh_W5NxBMR882F3lW83mhdT8jVN83MwydZZDT_X9W81pmJ72zV0jRf8nFFLg04 )\\r\\n\\r\\neichenwerk-gutschein-eichenwerk-eichenwerk-687131.png (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w0-T5jMY8W50kH_H6lZ3pmVQVYsc3lzYkfW6Dpc-Z1Wx19pW1wjDgC1wqQSYW2rK5Qt1tR2-MVfZV5M4HjTvBW3yp_Z37PSSlFW2Z1xpB1zT7JYW6HWYSd744GwpW6rl_-k2pRwjVW8pJqZQ4w0R-jW4Kd5hd1x28bkW4_cm424dD9xXN4Xt8yNYxbMwW7wX4lt5dttfyW2TgRl652VGsGN5pstS9ZZN2lW5WrQCV3gmJfvW6Gtbb53vRTdlW6ytJJT1Nxd8_W5gkLC75Zqg98W381ylz89WysgW7CJtcv5DhPV0W8hf6Vf4qDDdTN78_gXg_Ql2cW5fFTPj1F6s2MW7XnGlc6PkVbhN8dnqBJ2sQ_KW3BR6rl4BsrdrVFx9Gz2lRLbXW761M8Q5kHy7hW1bLm2d30qPPHW5Xwg7p5FKxL3f1PSV2004 )\\r\\n\\r\\nGutschein\\r\\n\\r\\nVerschenke Freude und Flexibilität mit unseren individuell anpassbaren Wertgutscheinen!\\r\\n\\r\\nMEHR ERFAHREB\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w0-T5jMY8W50kH_H6lZ3pmVQVYsc3lzYkfW6Dpc-Z1Wx19pW1wjDgC1wqQSYW2rK5Qt1tR2-MVfZV5M4HjTvBW3yp_Z37PSSlFW2Z1xpB1zT7JYW6HWYSd744GwpW6rl_-k2pRwjVW8pJqZQ4w0R-jW4Kd5hd1x28bkW4_cm424dD9xXN4Xt8yNYxbMwW7wX4lt5dttfyW2TgRl652VGsGN5pstS9ZZN2lW5WrQCV3gmJfvW6Gtbb53vRTdlW6ytJJT1Nxd8_W5gkLC75Zqg98W381ylz89WysgW7CJtcv5DhPV0W8hf6Vf4qDDdTN78_gXg_Ql2cW5fFTPj1F6s2MW7XnGlc6PkVbhN8dnqBJ2sQ_KW3BR6rl4BsrdrVFx9Gz2lRLbXW761M8Q5kHy7hW1bLm2d30qPPHW5Xwg7p5FKxL3f1PSV2004 )\\r\\n\\r\\nFacebook (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3kMW8LQXLF64PgNTN58Fh0_9jltgW68Vk_S9kg5y5N6TgQzwj56HTN8HFDZyd87_KW84KVlK8351WmW6dm01_1zCP2CMLc-0Gml40pW3M328Z7k10ZJW8Lr1Zq4wHvfQW8ZnQ3v8ZNFybN64KjJhqdHdBVGTldJ4L63MWN5GJmH8MYwvWW65v9rc709MkhW5ryxty2PPSK7MdYtg7KtnNzN17Dtp_ndD6sW1JMDTk5nrPyzW1QC_ls5kCmZSMYHQdWX6fJQW2XZnmJ9jjtnqW4Pjxzw97YZM3N7K5Bf65NqXkf16t_F804 )\\r\\n\\r\\nInstagram (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w0_M3lcq-W6N1vHY6lZ3ksN2ldlZrkfgchVdmmmy11ZSRsW6Ql5sM8VM61dW4rSvPb5zdfykW9b4D555bMSH_W7nBKnt7NmxpjW3B-7vz2XlxQyVDx5cB5Kr3csW4_l-l-2WHrBcV1Ftl924xLBSVFJ-mJ7DCCNyW61wyxj5nHPXJW21l-sF7dhF3hW2nt-3Y2F98l6N4kKDrWp6HFcW7HZnY67vBt9yW5PJF5H6s_ZHZVqq-5_3gp2V0N7D4TLC_0m17W4DDq7Q2dCmGmW5rJRyy8nkwwQW1dW4sh7fYb2Bf6pt6kP04 )\\r\\n\\r\\nEICHENWERK e.U., Hirschentanzstraße 11, Breitenfurt, Niederösterreich 2384, Österreich, 068110424342\\r\\n\\r\\nAbmelden (https://hs-25357905.s.hubspotstarter-eu1.net/preferences/de/unsubscribe?data=W2nVjwf3Y2wWyW4hb04t3bhLVcW3FgBNs1Z7xFqW1LHznr3C89gfW2RPq-Y4pBqZZW21sZzl4kwJCCW49zC4r3SYM58W3Vwz9T36jGz_W1NjQKk3jgpwCW45S4Z734xWGnW24-hd82KYV9WW2WGQpR1QdrFnW38mMcf1Q5kQQW3XZBWw3M9nmHW3G_n5K4psHW4W329Bm-4fsm6fW2MBYC843P0wyW1Qm6hk1VcCH2W2F-cQn309ndlW3yXvjK3g26HrW1ZqpzK1_c3GjW4kG6hh3yYfd5W1Zrl8934yJ49W1_l67-38fslYW2Hq0sQ2xKGBxW211b2M43qsbjW2FZLnV2FV92BW34CHPc3SJQWlW3gdwl03jf5-bW32CSYr1_lvwTW49Nnkm49LhpTW2FDp5l4rq3FtW3ZYt7W3yRqdDW4kKxtp4msYC-W2HtjJg49BL-bW3SDbX-25hV75W24TxVF3ZwC-3W1Vy5pd2qM6-HW3QNP_34pH00yW3yNXnP41JbGHW2xFYg73gccxlW2PyHL41LldsYW1_62-Z2PLdB4W3650sT49DW_xW2Kz8nS3DLXbrW2zWpTS2MmyVGW41zxmB3z8KdVW3LBKgh4tx2BWW47Dg873by_5VW38kjkD47v83MW43HzXP2HVQ0yW3K3L1w4rF2GsW3_zLV43DYf5ZW4mksjL4pjT9JW3GZj8S2xWGjLW4pqjxG21bpR10\\u0026_hsenc=p2ANqtz--upTI4xtxh18q0Tuk9UdY357sy_UsWGU3g9kT3Mb8mrzvVDgSqHzLd4wdCxC7e5ULgXqp1Z3P7aEcAcp12W5UHWlVp-A\\u0026_hsmi=129513735 )\\r\\nEinstellungen verwalten (https://hs-25357905.s.hubspotstarter-eu1.net/preferences/de/manage?data=W2nVjwf3Y2wWyW4hb04t3bhLVcW3FgBNs1Z7xFqW1LHznr3C89gfW2RPq-Y4pBqZZW21sZzl4kwJCCW49zC4r3SYM58W3Vwz9T36jGz_W1NjQKk3jgpwCW45S4Z734xWGnW24-hd82KYV9WW2WGQpR1QdrFnW38mMcf1Q5kQQW3XZBWw3M9nmHW3G_n5K4psHW4W329Bm-4fsm6fW2MBYC843P0wyW1Qm6hk1VcCH2W2F-cQn309ndlW3yXvjK3g26HrW1ZqpzK1_c3GjW4kG6hh3yYfd5W1Zrl8934yJ49W1_l67-38fslYW2Hq0sQ2xKGBxW211b2M43qsbjW2FZLnV2FV92BW34CHPc3SJQWlW3gdwl03jf5-bW32CSYr1_lvwTW49Nnkm49LhpTW2FDp5l4rq3FtW3ZYt7W3yRqdDW4kKxtp4msYC-W2HtjJg49BL-bW3SDbX-25hV75W24TxVF3ZwC-3W1Vy5pd2qM6-HW3QNP_34pH00yW3yNXnP41JbGHW2xFYg73gccxlW2PyHL41LldsYW1_62-Z2PLdB4W3650sT49DW_xW2Kz8nS3DLXbrW2zWpTS2MmyVGW41zxmB3z8KdVW3LBKgh4tx2BWW47Dg873by_5VW38kjkD47v83MW43HzXP2HVQ0yW3K3L1w4rF2GsW3_zLV43DYf5ZW4mksjL4pjT9JW3GZj8S2xWGjLW4pqjxG21bpR10\\u0026_hsenc=p2ANqtz--upTI4xtxh18q0Tuk9UdY357sy_UsWGU3g9kT3Mb8mrzvVDgSqHzLd4wdCxC7e5ULgXqp1Z3P7aEcAcp12W5UHWlVp-A\\u0026_hsmi=129513735 )\",\"labels\":[\"CATEGORY_PROMOTIONS\",\"UNREAD\",\"INBOX\"],\"headers\":{\"ARC-Authentication-Results\":\"i=1; mx.google.com;       dkim=pass header.i=@bf01.eu1.hubspotstarter.net header.s=hs1 header.b=NNiaVg+x;       dkim=pass header.i=@eichenwerk.at header.s=hs1-25357905 header.b=LZV9evDi;       spf=pass (google.com: domain of 1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net designates 143.244.84.161 as permitted sender) smtp.mailfrom=\\\"1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net\\\";       dmarc=pass (p=NONE sp=NONE dis=NONE) header.from=eichenwerk.at\",\"ARC-Message-Signature\":\"i=1; a=rsa-sha256; c=relaxed/relaxed; d=google.com; s=arc-20240605;        h=feedback-id:list-unsubscribe-post:precedence:mime-version:subject         :message-id:to:reply-to:from:date:list-unsubscribe:dkim-signature         :dkim-signature;        bh=lez4hoGAC9cA8VzrvkDFldkIbUUvPzE7jkKugidr2e4=;        fh=3jTgowhSmX/Yxq6vZoZ8ZaBHSO1O3L4jxXHHrreJdhw=;        b=T8BdJgFyiN4tCY8buWc+lSjCwL4qmUtkeqne2CTexEewgtsldM9b85q+reQW+uyk39         pAGCtvzt5lvQDSpssJwrPOUk0o8JtppTeDvmXrT7qETeKuGXp9XBEu0hailqBBBm2GVE         TXFw27NhQR92SZ8dA4LuwNb32rGQ9EXA30HQtg4BOXsnXEoZGaHvfX2oe3QnNhkikYrs         DDrK9B1LBJVT0nRVTERhI/J0gzPZ1k25pSxJV6BHrSTkZUJPhgY8VMr56aBhEaDKiZBa         VpJkl8pXavxuSsGY7bR1CRr5k8zxIErbX7ze37XoxSAIGHxhwPGkQRu9wImsrIg2Y6gR         T6ag==;        dara=google.com\",\"ARC-Seal\":\"i=1; a=rsa-sha256; t=1772000574; cv=none;        d=google.com; s=arc-20240605;        b=JGliZITpkdsrGusnwSX5fJItft82f4cJGe+s/d86Qc7IQLGXH7+VvgrrxLsbkQQdCP         CeaXTMf5uUdLj6N5Ir6IZjBGvB1I27xg0oQZuyrwCPrAg86n7Pz6xErQiRXOTr31YO+c         MuIG5cuvRNzcX7hSqhsaDFw7VJG2iQXIHYmCRkfZN9Lm8BPLIFKULQlzpuPHu8E5ks3z         Mr7qvZ0SymtomgiyDJ7q1CLm5Vvj5pGv5miIV4aDoLQdnVwOY8tqhBM60d6IZgOMne6o         OgGU3V1TB/8wwIlqfdtpXxZ4PTsnoOFA0ha7r4fIPWYpiJJvgZcACn/j4gz0vIMD6Imj         7xug==\",\"Authentication-Results\":\"mx.google.com;       dkim=pass header.i=@bf01.eu1.hubspotstarter.net header.s=hs1 header.b=NNiaVg+x;       dkim=pass header.i=@eichenwerk.at header.s=hs1-25357905 header.b=LZV9evDi;       spf=pass (google.com: domain of 1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net designates 143.244.84.161 as permitted sender) smtp.mailfrom=\\\"1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net\\\";       dmarc=pass (p=NONE sp=NONE dis=NONE) header.from=eichenwerk.at\",\"Content-Type\":\"multipart/alternative; boundary=\\\"----=_Part_371339_978480836.1772000574411\\\"\",\"DKIM-Signature\":\"v=1; s=hs1-25357905; d=eichenwerk.at;         i=@eichenwerk.at;         h=sender:from:from:reply-to:to:to:cc:cc:subject:subject:list-unsubscribe:form-sub:feedback-id:list-unsubscribe-post;         a=rsa-sha256; c=relaxed/relaxed;         bh=lez4hoGAC9cA8VzrvkDFldkIbUUvPzE7jkKugidr2e4=;         b=LZV9evDicjJML0GkeaVn/iWfOWUBatMJxrnlDo7RpeIMQugN9SWJH1yoVPKVbp         ZGaVWMjz6Apif8TSlyfSkvl5JcDdqbpimbp3XtHz2FPmn+HmWgsnYEuxG/vZlc/         ff0q6jeXH14bLfCQn3z4Ik0n67I2mMkVfo6WWg/cua3WOIXXHW40zA/+iFy+IAt         jFaYGMm496W1v+UZTYNjuOzBSFnJb8vUEmrxGXxLreNfFD6TUB0/s2frYYHxZuN         JLRWCISUtveT2jlKKd6DX/8UpicbBYKJLg5cq+wxtbWMqDsrPWimiuL78DkS4mg         JLtshh3nLzwvAdTTQjeNVouVIQYw==; q=dns/txt; t=1772000574;         x=1772263374;\",\"Date\":\"Wed, 25 Feb 2026 07:22:54 +0100\",\"Delivered-To\":\"reneheinzl@gmail.com\",\"Feedback-ID\":\"aepdskj:aicfd7p1:aidf:HubSpot\",\"From\":\"EICHENWERK \\u003coffice@eichenwerk.at\\u003e\",\"List-Unsubscribe\":\"\\u003cmailto:1axcumjdiysxvb9n1q0ugfyzsjkihdf2suloww-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net?subject=unsubscribe\\u003e, \\u003chttps://hs-25357905.s.hubspotstarter-eu1.net/subscription-preferences/v2/unsubscribe-all?data=W2nVjwf3Y2x4cW1QnmPS3ZXfGwW1ZkW7l43zkL9W3g6dK849M32kW2zJZ__1L8FY3W3QPxzY2HZYPcW4hs4K149slPgW1ZwX0r3K4ct8W4fpTxg4cPqBKW49BRVY2Wrcn1W3ZDSFf1L9zT-W36FY9_3SPQNrW41C_xp3M7jf3W2zKpZz2CTxbVW4cQGv13h-wqGW41DW523dpr6YW1Qm_Hn2WLWb5W47C2zL2CD9f-W1_b_xY1X6sZHW3_SLhZ4pmJLgW3ZTn-01LGQ6bW4hp_-z1LljHfW45VFw02xKbWNW2zTLm93ZtBBzW4mKdLw1XnBCJW34z-YN3SLW8NW34pN-g2FLjN6W4cHd5t3-0fqDW1NyjQ_2w37-3W1Bs3G_4fjNpCW3DMkf14tG70TW2Hsw7Y1LF_j-W3zb22J258j-QW2YykQm36m_5LW3c-87f21nkVqW1V9X5g1LwT4GW4hm-Jp3KcHkZW2zxtbt41DQCWW3XYL_143RtJbW2358412YmlcSW2HtL8h3_SLjnW3QQRG024ShRpW4rHB982Tgl75W1NnY582WKFTzW2TBpHK4cL78dW4pJFkC36gCmLW3M65bn2KXKBSW1BxJTy3LYZLfW3VHlft3BVY28W2FZmnT47VmjCW4knB4n4mtprCW4pylHY1X9Qc0W4ff0YC4fjsbbW30HSn72HPTt3W3LYZWf3z2WnHf2nD_vW04\\u003e\",\"List-Unsubscribe-Post\":\"List-Unsubscribe=One-Click\",\"MIME-Version\":\"1.0\",\"Message-ID\":\"\\u003c1772000572728.1465bea2-dcf3-4902-a504-de0f75a40f83@bf01.eu1.hubspotstarter.net\\u003e\",\"Precedence\":\"bulk\",\"Received\":\"by 172.16.38.52 with SMTP id a089896hoz74522ffiphsmac28zz1brbr8rb1kykln;        Wed, 25 Feb 2026 06:22:54 GMT\",\"Received-SPF\":\"pass (google.com: domain of 1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net designates 143.244.84.161 as permitted sender) client-ip=143.244.84.161;\",\"Reply-To\":\"office@eichenwerk.at\",\"Return-Path\":\"\\u003c1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net\\u003e\",\"Subject\":\"Der Frühling kommt mit neuen Terminen! \",\"To\":\"reneheinzl@gmail.com\",\"X-HS-Cid\":\"1axfanp4mrbz4cmkfutffu15eigxtiailgmstc\",\"X-Received\":\"by 2002:a05:6000:220b:b0:435:a194:734 with SMTP id ffacd0b85a97d-4398fa5ed3fmr1649380f8f.5.1772000574993;        Tue, 24 Feb 20\n... (truncated)",
    "is_error": false
  },
  {
    "name": "drive_search",
    "input": {
      "query": "modifiedDate \u003e= '2026-02-18'"
    },
    "output": "{\"error\":\"bad request: Invalid Value\"}",
    "is_error": true
  }
]
```

**Final Response:**
```
The error "bad request: Invalid Value" suggests there was an issue with the parameters provided in your request. Could you please clarify which function you were trying to use (e.g., `drive_search`, `calendar_list`, etc.) and the arguments you provided? This will help identify the invalid parameter or value. For example:
- Did you use a wrong date format (e.g., `YYYY-MM-DD`)?
- Did you reference a non-existent spreadsheet ID or calendar?
- Were there typos in parameter names (e.g., `query` vs `q`)? 

Let me know the specifics!
```

#### `qwen3:8b` -- complex-project-overview [Fail]

**Tool Calls:**
```json
[
  {
    "name": "email_search",
    "input": {
      "query": "after:2026-02-20"
    },
    "output": "[{\"id\":\"19c93b206838483e\",\"thread_id\":\"19c93b206838483e\",\"subject\":\"Neues Passwort bei BefundPost.at\",\"from\":\"BefundPost \\u003cinfo@befundpost.at\\u003e\",\"to\":[\"\\u003creneheinzl@gmail.com\\u003e\"],\"date\":\"2026-02-25T08:27:23+01:00\",\"body\":\"Sehr geehrter Benutzer von BefundPost.at,\\r\\n\\r\\nSie möchten Ihr Passwort zurücksetzen?\\r\\nKlicken Sie auf den untenstehenden Link und wählen Sie ein neues Passwort.\\r\\nhttps://www.befundpost.at/ueber/support/code/8df461ce\\r\\n\\r\\nSollte der Link nicht funktionieren kopieren Sie diesen einfach in Ihren Webbrowser.\\r\\n\\r\\nMit freundlichen Grüssen,\\r\\n\\r\\nIhr Team von BefundPost.at\\r\\n\\r\\n\\r\\n\",\"labels\":[\"IMPORTANT\",\"CATEGORY_UPDATES\",\"INBOX\"],\"headers\":{\"ARC-Authentication-Results\":\"i=1; mx.google.com;       dkim=pass header.i=@smtpcorp.com header.s=a1-4 header.b=xF7PfqTD;       dkim=pass header.i=@befundpost.at header.s=s669783 header.b=g0EYqfGc;       spf=pass (google.com: domain of bt.6yge2obdjpot7a7=w626qnmx3lo6=53ntq6atgsxi14@em669783.befundpost.at designates 103.2.141.76 as permitted sender) smtp.mailfrom=\\\"bT.6yge2obdjpot7a7=w626qnmx3lo6=53ntq6atgsxi14@em669783.befundpost.at\\\"\",\"ARC-Message-Signature\":\"i=1; a=rsa-sha256; c=relaxed/relaxed; d=google.com; s=arc-20240605;        h=feedback-id:message-id:subject:date:mime-version:from:reply:to         :dkim-signature:dkim-signature;        bh=bTEx1Wbg7+gGW1cHFmDIuD4cW95iuCMHc+DWYLk8FcI=;        fh=3jTgowhSmX/Yxq6vZoZ8ZaBHSO1O3L4jxXHHrreJdhw=;        b=Qt1kzV1zzgdr8Kn5hY2jQQDCTQWd6e5PLqz86cYNzYkwzgS09uKDErhNxThoGD9ZfW         y2TzEu5grzvt1/PboexMcYY1w4AELqZo7j6LXotbGnJPrZicGn1lzpXPAuphoDbujUwf         ka8Eoc8sVlvoyScDwmCvdMIAVO3/OjopVB2I0rBv3oq+cU99pme1Xgj5J+oeaKGHGxJn         4b/sBWNKbPyEl+5lqpa693yRM2ns73Pu6GNHb/TrhtBr0eyR85rko1+y9PFH8394rMDz         PlNsHdakE2D0y6lda+nV55iIomCnvpbHOEacAtFuJpYEh6dH/JmXY6b7ZQvgKkOaWZjb         a3uQ==;        dara=google.com\",\"ARC-Seal\":\"i=1; a=rsa-sha256; t=1772004444; cv=none;        d=google.com; s=arc-20240605;        b=CKWF7jSufA3le/3Qj0gwdIJIw6cv6/nno0ddnqVAfMQQpMYnDMWccFT1uicaO5AfCJ         WmOccRlmbc82wGzpErM/x5P6I8BLqsF8yY3L2BmyjeuipTQcjffESBP8ruoBcWum3yZ3         Om5Je6nWAVg5qgwmYDZZvBUf8lGHWmq/2myJiJDEi+e6xtEGhwx5h3j9/k3vAHTzKUa7         Ph98vFIG/V5/Iyx0Q05NDCvFPhqWBHkBNcX2ssb5KNjUu+4XiH8GA1gyqvoOo6zSey8O         V+zxLv79aQv0A6NdQ4wzb7aNyyle6XsXMfp0U6DFqAO5uADM4Ux+JaXO+ERwNyPAFvU3         N+kw==\",\"Authentication-Results\":\"mx.google.com;       dkim=pass header.i=@smtpcorp.com header.s=a1-4 header.b=xF7PfqTD;       dkim=pass header.i=@befundpost.at header.s=s669783 header.b=g0EYqfGc;       spf=pass (google.com: domain of bt.6yge2obdjpot7a7=w626qnmx3lo6=53ntq6atgsxi14@em669783.befundpost.at designates 103.2.141.76 as permitted sender) smtp.mailfrom=\\\"bT.6yge2obdjpot7a7=w626qnmx3lo6=53ntq6atgsxi14@em669783.befundpost.at\\\"\",\"Content-Type\":\"multipart/alternative; boundary=\\\"====1772004443====\\\"\",\"DKIM-Signature\":\"v=1; a=rsa-sha256; c=relaxed/relaxed; d=befundpost.at; i=@befundpost.at; q=dns/txt; s=s669783; t=1772004443; h=from : subject : to : message-id : date; bh=bTEx1Wbg7+gGW1cHFmDIuD4cW95iuCMHc+DWYLk8FcI=; b=g0EYqfGcbXJMhs9674je6G/W3AaUfD/l709vtSKjFJD2WGUOkffyqr/EGEyPDCNykwVnb 76+vGlWF09R/vgP4XuojuuQ4kBA4l05RWRkOYLSqdqr3ppkK+KVT7tN5f3pgQBbhxCt2qWV CsLQSxtOb65gZWeLuUWL1rO1ZQfJYuVZJjYe9mVRywfEm2XLA0vHXlhx1cpaEEzQNmWgUoK Tql+EC5KQ1xAJSpgnooQwhIamSs20HQLNB3O38R2cnSYYXiMdEoP38CoJ4bi3h4B9KSxZ4m aMfAyC+B+oRffCmUudnTLSl3jMzWB441gCeVzr12WdGlIs+ODtkNh2DEsqVw==\",\"Date\":\"Wed, 25 Feb 2026 08:27:23 +0100\",\"Delivered-To\":\"reneheinzl@gmail.com\",\"Feedback-ID\":\"669783m:669783aH0QqkA:669783sSvup15pkZ\",\"From\":\"BefundPost \\u003cinfo@befundpost.at\\u003e\",\"Message-Id\":\"\\u003cE1vv9Ip-AIkwcC8tGTf-Hk9B@message-id.smtpcorp.com\\u003e\",\"Mime-Version\":\"1.0\",\"Received\":\"from [10.131.241.105] (helo=steinhausen) by smtpcorp.com with esmtpa (Exim 4.99.1-S2G) (envelope-from \\u003cinfo@befundpost.at\\u003e) id 1vv9Ip-AIkwcC8tGTf-Hk9B for reneheinzl@gmail.com; Wed, 25 Feb 2026 07:27:23 +0000\",\"Received-SPF\":\"pass (google.com: domain of bt.6yge2obdjpot7a7=w626qnmx3lo6=53ntq6atgsxi14@em669783.befundpost.at designates 103.2.141.76 as permitted sender) client-ip=103.2.141.76;\",\"Reply\":\"BefundPost\\u003cinfo@befundpost.at\\u003e\",\"Return-Path\":\"\\u003cbT.6yge2obdjpot7a7=w626qnmx3lo6=53ntq6atgsxi14@em669783.befundpost.at\\u003e\",\"Subject\":\"Neues Passwort bei BefundPost.at\",\"To\":\"reneheinzl@gmail.com\",\"X-Bp2020-Id\":\"1067347\",\"X-Bp2020-Mode\":\"production\",\"X-Bp2020-Type\":\"sendPassword\",\"X-Received\":\"by 2002:a05:6000:2410:b0:439:9015:a96c with SMTP id ffacd0b85a97d-4399015ab62mr1767586f8f.35.1772004444054;        Tue, 24 Feb 2026 23:27:24 -0800 (PST)\",\"X-Report-Abuse\":\"Please forward a copy of this message, including all headers, to \\u003cabuse-report@smtp2go.com\\u003e\",\"X-Smtpcorp-Track\":\"odp-4OjZEfpl.5cQze8PvS2XV.z0GAYAgFs8M\"}},{\"id\":\"19c93b0b0050b332\",\"thread_id\":\"19c93b0b0050b332\",\"subject\":\"Ihre Benutzerkennungen bei BefundPost.at\",\"from\":\"BefundPost \\u003cinfo@befundpost.at\\u003e\",\"to\":[\"\\u003creneheinzl@gmail.com\\u003e\"],\"date\":\"2026-02-25T08:25:55+01:00\",\"body\":\"Guten Tag\\r\\n\\r\\nGerne teilen wir Ihnen die über Ihre E-Mail-Adresse registrierten Benutzer mit:\\r\\n\\r\\nIHR Labor - HEINZL RENE - Benutzerkennung: gjbo019\\r\\n\\r\\n\\r\\nPasswort vergessen? https://www.befundpost.at/ueber/support/?password\\r\\n\\r\\nFreundliche Grüsse\\r\\nIhr Team von BefundPost.at\\r\\n\\r\\n\\r\\n\",\"labels\":[\"IMPORTANT\",\"CATEGORY_UPDATES\",\"INBOX\"],\"attachments\":[{\"id\":\"ANGjdJ844MkmQdOjz3p8odmGh40m7qJLl-r6dVcYUJZlWSKWEzEqFBGItro2q77janMXtM_zhvg1izNeynOhDa-oEaYQkjr9XWT3UCIjR6PtpnBH_wkHKN8Z6XVNqKr5gyWDTWLqFZBFZ3eDYuKKDYO6YvQgHxcFDgl_2PWq-1p6gjUyOtlLojJ5_mt2U4kIbReJg9ZaLPBt6aFOdwjE9MvdKMqxJCZBwBfCOEG-rXTSFxU141LF34F5nPvU5GbZQldtuZ9aWSzAGXFXIFSy2ySO40ODHB0XM7QcI-ypGSZfTFQKQEGYvd6-kW2p9T76nG7ZFEq4n4LlNuL8a26-bOibncjK-IMzpJIMDI4SbXARVlzf2LcnqMdILq6npanVOpRYF6bwI9LjlrEfFsaVvBNGdu4Aa3Pxpby4NG_cpw\",\"filename\":\"BPCard_HEINZL_RENE_IHRLabor.pdf\",\"mime_type\":\"application/pdf\",\"size\":256642}],\"headers\":{\"ARC-Authentication-Results\":\"i=1; mx.google.com;       dkim=pass header.i=@smtpcorp.com header.s=a1-4 header.b=l4bpbanX;       dkim=pass header.i=@befundpost.at header.s=s669783 header.b=cuINsvtq;       spf=pass (google.com: domain of bt.8fqfstp6mo8n9f6=hoi6avxkdblx=hpyiw3ceaa3tqa@em669783.befundpost.at designates 103.2.141.76 as permitted sender) smtp.mailfrom=\\\"bT.8fqfstp6mo8n9f6=hoi6avxkdblx=hpyiw3ceaa3tqa@em669783.befundpost.at\\\"\",\"ARC-Message-Signature\":\"i=1; a=rsa-sha256; c=relaxed/relaxed; d=google.com; s=arc-20240605;        h=feedback-id:message-id:reply:subject:date:from:mime-version:to         :dkim-signature:dkim-signature;        bh=CqPMHwtlkw2UU7XXoS2JwxY51Fc8GRv1ixBOymnCcn8=;        fh=3jTgowhSmX/Yxq6vZoZ8ZaBHSO1O3L4jxXHHrreJdhw=;        b=PfNP3V4YP8o06a+zqZR5p5UG+YjvZSiI2c5Q0J2BhVugQFnTcqAx6mR2Y+Tf9rYTWy         UaR87Pkw4OVQZr4vg3j8P9KM1e2pP9DWt2JBTJrI5bnCYvocWHsTV0IkO0yaHjIWD9r+         DKUZ6w4WwKvfeFtQ2bibo2fGPSGxGyfpzNp6wGLaOdhTod9o1WKeMVHhv8GwtTFIms+U         zjU2JaYzbvSqDtIhgOOLpjLd9I+MkT1Gcdy4g70yW1Upq/29ebR2fdl9RGNkqn3nYcfm         SIGNDn36vnREy9HjE+fQwO8Al2lb+dgQ3tYrngi5J8ey5w6UUFgCko0W9u3+CSexfAha         mpLg==;        dara=google.com\",\"ARC-Seal\":\"i=1; a=rsa-sha256; t=1772004356; cv=none;        d=google.com; s=arc-20240605;        b=FhMJv5lXgnEX7zzHcsecdy0Qo0bAo29xbsYxx+xIVQ4IXxjn9o1k+pFaKDuCKzsn0+         /d20RLJLjo4c/CErAAf2DOi6ba1rIJ1kxibU+S8SlkxoNKD7LGOtH2yoRUegjEdvETM0         23phVSPpx8wslXTLhKaeeVHQKYwwcJI5XUKgJYvOzJfHtd6xndmwwoZoNNI7ElYw/Wyg         I04Lu9/t9mWfg9f/m9bvsSgqg2VX3NtunOAEsgN6dpjw0Edg2pl5mmNiApv3sZAmRxc7         M8rbo7xCEYVTjIkcwgT+GEXZd79mZp19J8zKCt3uzYdP2tA0xn2hRc6wex3xljpoWrNn         KTjQ==\",\"Authentication-Results\":\"mx.google.com;       dkim=pass header.i=@smtpcorp.com header.s=a1-4 header.b=l4bpbanX;       dkim=pass header.i=@befundpost.at header.s=s669783 header.b=cuINsvtq;       spf=pass (google.com: domain of bt.8fqfstp6mo8n9f6=hoi6avxkdblx=hpyiw3ceaa3tqa@em669783.befundpost.at designates 103.2.141.76 as permitted sender) smtp.mailfrom=\\\"bT.8fqfstp6mo8n9f6=hoi6avxkdblx=hpyiw3ceaa3tqa@em669783.befundpost.at\\\"\",\"Content-Type\":\"multipart/mixed; boundary=\\\"====1772004354====\\\"\",\"DKIM-Signature\":\"v=1; a=rsa-sha256; c=relaxed/relaxed; d=befundpost.at; i=@befundpost.at; q=dns/txt; s=s669783; t=1772004356; h=from : subject : to : message-id : date; bh=CqPMHwtlkw2UU7XXoS2JwxY51Fc8GRv1ixBOymnCcn8=; b=cuINsvtqAIHZzil3MIK6cRVPhLMsMbA4tYj06NA1GvcLwVKX6fHDDxIRVHNqvGnRVXNaz EkilDFYavEHF1PBQAWzS1SxFH9FwL1T8WlpRjyHqe/WxX+J2JDclqZwcmFfHTKxC5bkHUsl qf1FrgSUVyjovP4PNg9edehnnl0XCkGlRHvJFeuGoZC1fMfwTDM2aVkMpDb2NnvAQP8W5aU fa3MmZhhclVY+tniywK/PZcmtXfy5zbMjasHSzCwyhCgw709eGAXnceYSdWupeKO6O151Ha /kR5YWhlc4D4nvvpKIvUsq1BNNF+vNkBg1RVDVL0UGAPs00PkqgV42FDheiQ==\",\"Date\":\"Wed, 25 Feb 2026 08:25:55 +0100\",\"Delivered-To\":\"reneheinzl@gmail.com\",\"Feedback-ID\":\"669783m:669783aH0QqkA:669783s9l-54EwA1\",\"From\":\"BefundPost \\u003cinfo@befundpost.at\\u003e\",\"Message-Id\":\"\\u003cE1vv9HP-FnQW0hPotsf-lJiH@message-id.smtpcorp.com\\u003e\",\"Mime-Version\":\"1.0\",\"Received\":\"from [10.131.241.105] (helo=steinhausen) by smtpcorp.com with esmtpa (Exim 4.99.1-S2G) (envelope-from \\u003cinfo@befundpost.at\\u003e) id 1vv9HP-FnQW0hPotsf-lJiH for reneheinzl@gmail.com; Wed, 25 Feb 2026 07:25:55 +0000\",\"Received-SPF\":\"pass (google.com: domain of bt.8fqfstp6mo8n9f6=hoi6avxkdblx=hpyiw3ceaa3tqa@em669783.befundpost.at designates 103.2.141.76 as permitted sender) client-ip=103.2.141.76;\",\"Reply\":\"BefundPost\\u003cinfo@befundpost.at\\u003e\",\"Return-Path\":\"\\u003cbT.8fqfstp6mo8n9f6=hoi6avxkdblx=hpyiw3ceaa3tqa@em669783.befundpost.at\\u003e\",\"Subject\":\"Ihre Benutzerkennungen bei BefundPost.at\",\"To\":\"reneheinzl@gmail.com\",\"X-Bp2020-Mode\":\"production\",\"X-Bp2020-Type\":\"sendUsers\",\"X-Received\":\"by 2002:a5d:5f47:0:b0:437:6c1f:de27 with SMTP id ffacd0b85a97d-4398faefc7dmr1985630f8f.37.1772004356325;        Tue, 24 Feb 2026 23:25:56 -0800 (PST)\",\"X-Report-Abuse\":\"Please forward a copy of this message, including all headers, to \\u003cabuse-report@smtp2go.com\\u003e\",\"X-Smtpcorp-Track\":\"ifQmzw-xrXvA.INe-MYtwwvSL.t99KwAU_NgF\"}},{\"id\":\"19c9376fead35da9\",\"thread_id\":\"19c9376fead35da9\",\"subject\":\"Der Frühling kommt mit neuen Terminen! \",\"from\":\"EICHENWERK \\u003coffice@eichenwerk.at\\u003e\",\"to\":[\"\\u003creneheinzl@gmail.com\\u003e\"],\"date\":\"2026-02-25T07:22:54+01:00\",\"body\":\"Runter von der Couch \\u0026 ab zur Range - Das heißt es mit dem kommenden Frühling und unseren neuen Terminen!  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­\\r\\n\\r\\nMit dem Frühling kommen auch die neuen Termine!\\r\\n\\r\\nRunter von der Couch \\u0026 ab zur Range! Das heißt es mit dem nahenden Frühling - der nicht nur gutes Wetter mit sich bringt, sondern auch unsere neuen Termine.\\r\\n\\r\\nHESA VERTEIDIGUNGSSCHIESSEN\\r\\n\\r\\nDas Basisseminar und Sondermodule.\\r\\n\\r\\nModul 1 bis 4  (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3m3W2RkTxx3JdwWzMjx7rP5xrLcVcVF7b2jcHYmW2Jk4y38mQ59BW9bQDNy5xFc9pW4G5Szn5JNgK4W2tc8bh8b3hgCW7tN_GG7GDZ5GW3w-9W33zvG3sW4S9tDW2h3twqW5JFjhW64J2y8W2V4p-Z83zBF3W2TFVJv5SgLJlW1snT-42W7jb6W1ZNQ8Y5PW0nKW713HzV1yg0mKN1lv7gK65VdrW7VhJq94d6Dz0N3NLrVXrNSbPW5mSGWQ6XJMCbVGSJgQ2lLPcVW849lN63D1QxFW4LbFpt3C_kfRW8pvfDd6jSKmvf1PSQD604 ) – sicheres Schießen von Anfang an, bis hin zum Schießen in der Bewegung und in Stresssituationen.\\r\\n\\r\\nModul 6 (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10Y3lcq-W95jsWP6lZ3mVW4FYyqF6xw9tPVnXfbt1vdM5pW21j2SS8fSG5CW44T1Lw6Lb39nW5KwsNJ7VsvY6W3kzyCG46BpQSW5fv4Kc7q8N4vW31fCCn5HdQgxVpQhdH7sM-Z6W3kswPl7qwb_nW6Q7Rl-7QHvJCW44zGfk1VsyL8W2yf3Kg7RFhwJVdylt35QqKnpW2D-2Nr3pTXsKW3TxLR35Hjc79W4-LHTF1QdfqtW4whrhL1DbL-1V-nnYc7Hq6BMW69Ghy06KS66FW3qY_q43q_d-kW27Vg3X1Xpx-tW5zHZ3_3fybjSVz9Wwn4zhv7CW8hzRkv46MGjHW249k5B28GRdhW6z0-ts2QvDSFW3-GmGD7VQks2W1zg5lP135_nyW6kKl6P8ssBtZf1kdCJF04 ) – Notwehrsituationen im Eigenheim: Szenarientraining, Videoanalyse und rechtliche Aufklärung.\\r\\n\\r\\nGrundschulung Gewehr  (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10Y3lcq-W95jsWP6lZ3pWW23JhYZ6pX-jTW5LQZx082mtPrMQSxR47XZz0VpgFY97lPJc5W10mzry3SMdTkW6T5WF94YGPlhW8h0S1g940WrxF6nMjn-kB9mW2DLNZs9dzKGWW6C7CYr2YvVTTW92LYj420wdtHW5rWtbq3JNV7pVLk-Dj7SkfNdN87YR8spsCtpW65PRK36rHFQqW7XjC4g800c_0VsYzj8586fMtW8BytpS6jd3RlW2cHR4R3yl-sgW95cCNc1wPHFnVf1gFg1n5-lmN985h_T8sRKmW6R8wSL9lLJMQW60xbBF4lkktPW4MBDfC4VYS4jN3NldCPr6jy7W5rb9Vk2tKFDvW8pr-4J8hrNVxW5dtTL47Q36hsW8T0HvG6GBzb_f28VMNl04 ) Langwaffenausbildungen für Anfänger (Grundschulung) und Fortgeschrittene (Modul 1-2-3)\\r\\n\\r\\nZU DEN KURSEN\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3pCW5W0TG_9dWky4N4Pn7KV5-FHTW2YHM5r2nn2YsVxjbgx5Jh4Y8W4wMjJx3fXXSGW2Dl5DK4sd3czW1zvJdh88TFh_W7Vktk74gDttJW38mSQ17Bh9trW8YRHV92KNNwyW3T7wrX98TyNzW4_TMCP3G-qkQW4yWy8Y4dp330W1KGZNW78fkdKW1McBLv55t0NNW6XfQQY6d2kZTW5rdsyY2vYcg5Vp0xxw8mwm7FW5dXvzW41CSsPN6frdYSDQPqlW1jhy0p71GBcFVLtV4j4fW4dbW1K_Rn_5n7VzJW11XVn63VXDWqf77VRFR04 )\\r\\n\\r\\nModul 9 Pistole.\\r\\n\\r\\nConcealed carry (verdeckte Trageweise) wir zeigen dir im Kurs unterschiedliche Präsentationstechniken, verschiedene Holster und trainieren direkt diverse Drills.\\r\\n\\r\\nMEHR ERFAHREN\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10F3lcq-W8wLKSR6lZ3kqW2GN4D73mW2GdW6KQTs64L7bHYW1-W8Sj7sWvJbW1GbHrH6rSgQMW8N8TWN6Rw3GSW5BFG-L8vt1MsW1D3wlC3f2WKZN1C64BFPQQFYW5Ksjn31M9S-nW20G-ts27L94wW17Ynpt6mnrxgW86HxXn2G4b0WVWVfWM6bMqXkM2-Jlvy6hbSV9xHtk73w6T-W1J-XvV3P4YJPW3T6M6v7c1FzqW7-VS7Q12g64nW2hPjSz3RHWRNW4h1mZq7Krr6MW2WYcP21BC085W38tk0Y3BZqfdW82LNMN89VydlW8zL0Rb6vpgFzV7VGjJ35QmFlW7Pkpx27ptWJgW1NgkLF83FMZZMh2rvbMCcVYf62j-fl04 )\\r\\n\\r\\nTraining mit Köpfchen.\\r\\n\\r\\nZum Fördern deiner Schießfertigkeit, Wahrnehmungs- und Konzentrationsfähigkeit!\\r\\n\\r\\nMEHR ERFAHREB\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10F3lcq-W8wLKSR6lZ3lWW3WWcP66fbBVNW6LQ--B854DbxW6d6xRs7cBfvFW8tP5ny88-dxtW7kkjN-20RgBBW4vQNx51Jz5KDW13y8kY7zpg6FVlNjR07DDRtCV6pmkB3KbdTPW4SK5X861SnK6W8L9VKp3KBtxhW2w1hVD4LqcQnW7NbjF069RyzgW45Ht2c4vpgLbN2t4tn5Fv6QMW3KxN2y67rdf7W91b2wp2k3tVbW6mMmMm60y5zxVkXL_R5hbWjZW1YBYYF1r14QqMx8Nd71vXz2W1ykJQq17lT74W86QJb34pBv0GVTfYws7tHNbgW2zMDl72yj-q1V5Ztmx9k0MpBN1vnjxLJ_qnnW5402mq2FDrzzf7R37Bl04 )\\r\\n\\r\\n1.png (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3m6VrB0Y12mn9NxW4fqL_Q4Tf0_kN3hjVbS-KyZZW307Gqk8ptZRhW671khG79bbkyW5n54Kt628q4lW8WsKlM26wlcJW1Fc-QX3kTq0MVZmWQQ4wG5z4W5X2-HZ5hLHGrW5XhNYq2B_xwwW99QqHC4gsbkvN27FYxz8V1KfW5PB_Gv3mg2yWW17bzz586vC8MW5KfvC03VWRG2W42yqDV72F0sYW7T7CyQ618Kt2W2mnKt58xpM92W6wdtQY8ykDh_W5NxBMR882F3lW83mhdT8jVN83MwydZZDT_X9W81pmJ72zV0jRf8nFFLg04 )\\r\\n\\r\\nSurvivalrally 2026\\r\\n\\r\\nSichere dir dein Ticket für die Survivalrally 2026 und werde Teil des größten Bushcraft \\u0026 Survival Events in Österreich.\\r\\n\\r\\nMEHR ERFAHREN\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cN\n... (truncated)",
    "is_error": false
  },
  {
    "name": "calendar_list",
    "input": {
      "calendar_id": "primary",
      "from": "2026-02-20",
      "to": "2026-02-27"
    },
    "output": "{\"error\":\"bad request: Bad Request\"}",
    "is_error": true
  },
  {
    "name": "drive_search",
    "input": {
      "query": "modifiedDate \u003e= '2026-02-20'"
    },
    "output": "{\"error\":\"bad request: Invalid Value\"}",
    "is_error": true
  }
]
```

**Final Response:**
```
It seems there was an issue with the tool calls. Let me help you troubleshoot:

1. **Check function parameters** - Ensure you're using the correct parameter names and values for each tool. For example:
   - `drive_ls` requires parameters like `max`, `parent`, or `query`
   - `email_search` needs a properly formatted query string (e.g., `from:alice subject:budget`)

2. **Verify syntax** - Make sure you're using the correct syntax for each tool's parameters. For example:
   - Google Drive queries use quotes around values: `"name contains 'budget'"`
   - Email searches use Gmail-style filters: `from:john is:unread`

3. **Check for typos** - Confirm the function names match exactly as defined (e.g., `drive_search` vs `drive_Search`)

Would you like to try a specific task again with these considerations in mind?
```

#### `qwen3:8b` -- complex-day-prep [Fail]

**Tool Calls:**
```json
[
  {
    "name": "calendar_list",
    "input": {
      "calendar_id": "primary",
      "from": "2026-02-25",
      "to": "2026-02-25"
    },
    "output": "{\"error\":\"bad request: Bad Request\"}",
    "is_error": true
  },
  {
    "name": "email_search",
    "input": {
      "query": "is:unread"
    },
    "output": "[{\"id\":\"19c9376fead35da9\",\"thread_id\":\"19c9376fead35da9\",\"subject\":\"Der Frühling kommt mit neuen Terminen! \",\"from\":\"EICHENWERK \\u003coffice@eichenwerk.at\\u003e\",\"to\":[\"\\u003creneheinzl@gmail.com\\u003e\"],\"date\":\"2026-02-25T07:22:54+01:00\",\"body\":\"Runter von der Couch \\u0026 ab zur Range - Das heißt es mit dem kommenden Frühling und unseren neuen Terminen!  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏  ͏ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­ ­\\r\\n\\r\\nMit dem Frühling kommen auch die neuen Termine!\\r\\n\\r\\nRunter von der Couch \\u0026 ab zur Range! Das heißt es mit dem nahenden Frühling - der nicht nur gutes Wetter mit sich bringt, sondern auch unsere neuen Termine.\\r\\n\\r\\nHESA VERTEIDIGUNGSSCHIESSEN\\r\\n\\r\\nDas Basisseminar und Sondermodule.\\r\\n\\r\\nModul 1 bis 4  (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3m3W2RkTxx3JdwWzMjx7rP5xrLcVcVF7b2jcHYmW2Jk4y38mQ59BW9bQDNy5xFc9pW4G5Szn5JNgK4W2tc8bh8b3hgCW7tN_GG7GDZ5GW3w-9W33zvG3sW4S9tDW2h3twqW5JFjhW64J2y8W2V4p-Z83zBF3W2TFVJv5SgLJlW1snT-42W7jb6W1ZNQ8Y5PW0nKW713HzV1yg0mKN1lv7gK65VdrW7VhJq94d6Dz0N3NLrVXrNSbPW5mSGWQ6XJMCbVGSJgQ2lLPcVW849lN63D1QxFW4LbFpt3C_kfRW8pvfDd6jSKmvf1PSQD604 ) – sicheres Schießen von Anfang an, bis hin zum Schießen in der Bewegung und in Stresssituationen.\\r\\n\\r\\nModul 6 (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10Y3lcq-W95jsWP6lZ3mVW4FYyqF6xw9tPVnXfbt1vdM5pW21j2SS8fSG5CW44T1Lw6Lb39nW5KwsNJ7VsvY6W3kzyCG46BpQSW5fv4Kc7q8N4vW31fCCn5HdQgxVpQhdH7sM-Z6W3kswPl7qwb_nW6Q7Rl-7QHvJCW44zGfk1VsyL8W2yf3Kg7RFhwJVdylt35QqKnpW2D-2Nr3pTXsKW3TxLR35Hjc79W4-LHTF1QdfqtW4whrhL1DbL-1V-nnYc7Hq6BMW69Ghy06KS66FW3qY_q43q_d-kW27Vg3X1Xpx-tW5zHZ3_3fybjSVz9Wwn4zhv7CW8hzRkv46MGjHW249k5B28GRdhW6z0-ts2QvDSFW3-GmGD7VQks2W1zg5lP135_nyW6kKl6P8ssBtZf1kdCJF04 ) – Notwehrsituationen im Eigenheim: Szenarientraining, Videoanalyse und rechtliche Aufklärung.\\r\\n\\r\\nGrundschulung Gewehr  (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10Y3lcq-W95jsWP6lZ3pWW23JhYZ6pX-jTW5LQZx082mtPrMQSxR47XZz0VpgFY97lPJc5W10mzry3SMdTkW6T5WF94YGPlhW8h0S1g940WrxF6nMjn-kB9mW2DLNZs9dzKGWW6C7CYr2YvVTTW92LYj420wdtHW5rWtbq3JNV7pVLk-Dj7SkfNdN87YR8spsCtpW65PRK36rHFQqW7XjC4g800c_0VsYzj8586fMtW8BytpS6jd3RlW2cHR4R3yl-sgW95cCNc1wPHFnVf1gFg1n5-lmN985h_T8sRKmW6R8wSL9lLJMQW60xbBF4lkktPW4MBDfC4VYS4jN3NldCPr6jy7W5rb9Vk2tKFDvW8pr-4J8hrNVxW5dtTL47Q36hsW8T0HvG6GBzb_f28VMNl04 ) Langwaffenausbildungen für Anfänger (Grundschulung) und Fortgeschrittene (Modul 1-2-3)\\r\\n\\r\\nZU DEN KURSEN\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3pCW5W0TG_9dWky4N4Pn7KV5-FHTW2YHM5r2nn2YsVxjbgx5Jh4Y8W4wMjJx3fXXSGW2Dl5DK4sd3czW1zvJdh88TFh_W7Vktk74gDttJW38mSQ17Bh9trW8YRHV92KNNwyW3T7wrX98TyNzW4_TMCP3G-qkQW4yWy8Y4dp330W1KGZNW78fkdKW1McBLv55t0NNW6XfQQY6d2kZTW5rdsyY2vYcg5Vp0xxw8mwm7FW5dXvzW41CSsPN6frdYSDQPqlW1jhy0p71GBcFVLtV4j4fW4dbW1K_Rn_5n7VzJW11XVn63VXDWqf77VRFR04 )\\r\\n\\r\\nModul 9 Pistole.\\r\\n\\r\\nConcealed carry (verdeckte Trageweise) wir zeigen dir im Kurs unterschiedliche Präsentationstechniken, verschiedene Holster und trainieren direkt diverse Drills.\\r\\n\\r\\nMEHR ERFAHREN\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10F3lcq-W8wLKSR6lZ3kqW2GN4D73mW2GdW6KQTs64L7bHYW1-W8Sj7sWvJbW1GbHrH6rSgQMW8N8TWN6Rw3GSW5BFG-L8vt1MsW1D3wlC3f2WKZN1C64BFPQQFYW5Ksjn31M9S-nW20G-ts27L94wW17Ynpt6mnrxgW86HxXn2G4b0WVWVfWM6bMqXkM2-Jlvy6hbSV9xHtk73w6T-W1J-XvV3P4YJPW3T6M6v7c1FzqW7-VS7Q12g64nW2hPjSz3RHWRNW4h1mZq7Krr6MW2WYcP21BC085W38tk0Y3BZqfdW82LNMN89VydlW8zL0Rb6vpgFzV7VGjJ35QmFlW7Pkpx27ptWJgW1NgkLF83FMZZMh2rvbMCcVYf62j-fl04 )\\r\\n\\r\\nTraining mit Köpfchen.\\r\\n\\r\\nZum Fördern deiner Schießfertigkeit, Wahrnehmungs- und Konzentrationsfähigkeit!\\r\\n\\r\\nMEHR ERFAHREB\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w10F3lcq-W8wLKSR6lZ3lWW3WWcP66fbBVNW6LQ--B854DbxW6d6xRs7cBfvFW8tP5ny88-dxtW7kkjN-20RgBBW4vQNx51Jz5KDW13y8kY7zpg6FVlNjR07DDRtCV6pmkB3KbdTPW4SK5X861SnK6W8L9VKp3KBtxhW2w1hVD4LqcQnW7NbjF069RyzgW45Ht2c4vpgLbN2t4tn5Fv6QMW3KxN2y67rdf7W91b2wp2k3tVbW6mMmMm60y5zxVkXL_R5hbWjZW1YBYYF1r14QqMx8Nd71vXz2W1ykJQq17lT74W86QJb34pBv0GVTfYws7tHNbgW2zMDl72yj-q1V5Ztmx9k0MpBN1vnjxLJ_qnnW5402mq2FDrzzf7R37Bl04 )\\r\\n\\r\\n1.png (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3m6VrB0Y12mn9NxW4fqL_Q4Tf0_kN3hjVbS-KyZZW307Gqk8ptZRhW671khG79bbkyW5n54Kt628q4lW8WsKlM26wlcJW1Fc-QX3kTq0MVZmWQQ4wG5z4W5X2-HZ5hLHGrW5XhNYq2B_xwwW99QqHC4gsbkvN27FYxz8V1KfW5PB_Gv3mg2yWW17bzz586vC8MW5KfvC03VWRG2W42yqDV72F0sYW7T7CyQ618Kt2W2mnKt58xpM92W6wdtQY8ykDh_W5NxBMR882F3lW83mhdT8jVN83MwydZZDT_X9W81pmJ72zV0jRf8nFFLg04 )\\r\\n\\r\\nSurvivalrally 2026\\r\\n\\r\\nSichere dir dein Ticket für die Survivalrally 2026 und werde Teil des größten Bushcraft \\u0026 Survival Events in Österreich.\\r\\n\\r\\nMEHR ERFAHREN\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3m6VrB0Y12mn9NxW4fqL_Q4Tf0_kN3hjVbS-KyZZW307Gqk8ptZRhW671khG79bbkyW5n54Kt628q4lW8WsKlM26wlcJW1Fc-QX3kTq0MVZmWQQ4wG5z4W5X2-HZ5hLHGrW5XhNYq2B_xwwW99QqHC4gsbkvN27FYxz8V1KfW5PB_Gv3mg2yWW17bzz586vC8MW5KfvC03VWRG2W42yqDV72F0sYW7T7CyQ618Kt2W2mnKt58xpM92W6wdtQY8ykDh_W5NxBMR882F3lW83mhdT8jVN83MwydZZDT_X9W81pmJ72zV0jRf8nFFLg04 )\\r\\n\\r\\neichenwerk-gutschein-eichenwerk-eichenwerk-687131.png (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w0-T5jMY8W50kH_H6lZ3pmVQVYsc3lzYkfW6Dpc-Z1Wx19pW1wjDgC1wqQSYW2rK5Qt1tR2-MVfZV5M4HjTvBW3yp_Z37PSSlFW2Z1xpB1zT7JYW6HWYSd744GwpW6rl_-k2pRwjVW8pJqZQ4w0R-jW4Kd5hd1x28bkW4_cm424dD9xXN4Xt8yNYxbMwW7wX4lt5dttfyW2TgRl652VGsGN5pstS9ZZN2lW5WrQCV3gmJfvW6Gtbb53vRTdlW6ytJJT1Nxd8_W5gkLC75Zqg98W381ylz89WysgW7CJtcv5DhPV0W8hf6Vf4qDDdTN78_gXg_Ql2cW5fFTPj1F6s2MW7XnGlc6PkVbhN8dnqBJ2sQ_KW3BR6rl4BsrdrVFx9Gz2lRLbXW761M8Q5kHy7hW1bLm2d30qPPHW5Xwg7p5FKxL3f1PSV2004 )\\r\\n\\r\\nGutschein\\r\\n\\r\\nVerschenke Freude und Flexibilität mit unseren individuell anpassbaren Wertgutscheinen!\\r\\n\\r\\nMEHR ERFAHREB\\r\\n(https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w0-T5jMY8W50kH_H6lZ3pmVQVYsc3lzYkfW6Dpc-Z1Wx19pW1wjDgC1wqQSYW2rK5Qt1tR2-MVfZV5M4HjTvBW3yp_Z37PSSlFW2Z1xpB1zT7JYW6HWYSd744GwpW6rl_-k2pRwjVW8pJqZQ4w0R-jW4Kd5hd1x28bkW4_cm424dD9xXN4Xt8yNYxbMwW7wX4lt5dttfyW2TgRl652VGsGN5pstS9ZZN2lW5WrQCV3gmJfvW6Gtbb53vRTdlW6ytJJT1Nxd8_W5gkLC75Zqg98W381ylz89WysgW7CJtcv5DhPV0W8hf6Vf4qDDdTN78_gXg_Ql2cW5fFTPj1F6s2MW7XnGlc6PkVbhN8dnqBJ2sQ_KW3BR6rl4BsrdrVFx9Gz2lRLbXW761M8Q5kHy7hW1bLm2d30qPPHW5Xwg7p5FKxL3f1PSV2004 )\\r\\n\\r\\nFacebook (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w1023lcq-W7lCdLW6lZ3kMW8LQXLF64PgNTN58Fh0_9jltgW68Vk_S9kg5y5N6TgQzwj56HTN8HFDZyd87_KW84KVlK8351WmW6dm01_1zCP2CMLc-0Gml40pW3M328Z7k10ZJW8Lr1Zq4wHvfQW8ZnQ3v8ZNFybN64KjJhqdHdBVGTldJ4L63MWN5GJmH8MYwvWW65v9rc709MkhW5ryxty2PPSK7MdYtg7KtnNzN17Dtp_ndD6sW1JMDTk5nrPyzW1QC_ls5kCmZSMYHQdWX6fJQW2XZnmJ9jjtnqW4Pjxzw97YZM3N7K5Bf65NqXkf16t_F804 )\\r\\n\\r\\nInstagram (https://d2-26S04.eu1.hubspotlinksstarter.com/Ctc/5F+113/d2-26S04/VVwCwn5ZltkVM5cNMlzw8vcW4PtzZh5KYCMpN22w0_M3lcq-W6N1vHY6lZ3ksN2ldlZrkfgchVdmmmy11ZSRsW6Ql5sM8VM61dW4rSvPb5zdfykW9b4D555bMSH_W7nBKnt7NmxpjW3B-7vz2XlxQyVDx5cB5Kr3csW4_l-l-2WHrBcV1Ftl924xLBSVFJ-mJ7DCCNyW61wyxj5nHPXJW21l-sF7dhF3hW2nt-3Y2F98l6N4kKDrWp6HFcW7HZnY67vBt9yW5PJF5H6s_ZHZVqq-5_3gp2V0N7D4TLC_0m17W4DDq7Q2dCmGmW5rJRyy8nkwwQW1dW4sh7fYb2Bf6pt6kP04 )\\r\\n\\r\\nEICHENWERK e.U., Hirschentanzstraße 11, Breitenfurt, Niederösterreich 2384, Österreich, 068110424342\\r\\n\\r\\nAbmelden (https://hs-25357905.s.hubspotstarter-eu1.net/preferences/de/unsubscribe?data=W2nVjwf3Y2wWyW4hb04t3bhLVcW3FgBNs1Z7xFqW1LHznr3C89gfW2RPq-Y4pBqZZW21sZzl4kwJCCW49zC4r3SYM58W3Vwz9T36jGz_W1NjQKk3jgpwCW45S4Z734xWGnW24-hd82KYV9WW2WGQpR1QdrFnW38mMcf1Q5kQQW3XZBWw3M9nmHW3G_n5K4psHW4W329Bm-4fsm6fW2MBYC843P0wyW1Qm6hk1VcCH2W2F-cQn309ndlW3yXvjK3g26HrW1ZqpzK1_c3GjW4kG6hh3yYfd5W1Zrl8934yJ49W1_l67-38fslYW2Hq0sQ2xKGBxW211b2M43qsbjW2FZLnV2FV92BW34CHPc3SJQWlW3gdwl03jf5-bW32CSYr1_lvwTW49Nnkm49LhpTW2FDp5l4rq3FtW3ZYt7W3yRqdDW4kKxtp4msYC-W2HtjJg49BL-bW3SDbX-25hV75W24TxVF3ZwC-3W1Vy5pd2qM6-HW3QNP_34pH00yW3yNXnP41JbGHW2xFYg73gccxlW2PyHL41LldsYW1_62-Z2PLdB4W3650sT49DW_xW2Kz8nS3DLXbrW2zWpTS2MmyVGW41zxmB3z8KdVW3LBKgh4tx2BWW47Dg873by_5VW38kjkD47v83MW43HzXP2HVQ0yW3K3L1w4rF2GsW3_zLV43DYf5ZW4mksjL4pjT9JW3GZj8S2xWGjLW4pqjxG21bpR10\\u0026_hsenc=p2ANqtz--upTI4xtxh18q0Tuk9UdY357sy_UsWGU3g9kT3Mb8mrzvVDgSqHzLd4wdCxC7e5ULgXqp1Z3P7aEcAcp12W5UHWlVp-A\\u0026_hsmi=129513735 )\\r\\nEinstellungen verwalten (https://hs-25357905.s.hubspotstarter-eu1.net/preferences/de/manage?data=W2nVjwf3Y2wWyW4hb04t3bhLVcW3FgBNs1Z7xFqW1LHznr3C89gfW2RPq-Y4pBqZZW21sZzl4kwJCCW49zC4r3SYM58W3Vwz9T36jGz_W1NjQKk3jgpwCW45S4Z734xWGnW24-hd82KYV9WW2WGQpR1QdrFnW38mMcf1Q5kQQW3XZBWw3M9nmHW3G_n5K4psHW4W329Bm-4fsm6fW2MBYC843P0wyW1Qm6hk1VcCH2W2F-cQn309ndlW3yXvjK3g26HrW1ZqpzK1_c3GjW4kG6hh3yYfd5W1Zrl8934yJ49W1_l67-38fslYW2Hq0sQ2xKGBxW211b2M43qsbjW2FZLnV2FV92BW34CHPc3SJQWlW3gdwl03jf5-bW32CSYr1_lvwTW49Nnkm49LhpTW2FDp5l4rq3FtW3ZYt7W3yRqdDW4kKxtp4msYC-W2HtjJg49BL-bW3SDbX-25hV75W24TxVF3ZwC-3W1Vy5pd2qM6-HW3QNP_34pH00yW3yNXnP41JbGHW2xFYg73gccxlW2PyHL41LldsYW1_62-Z2PLdB4W3650sT49DW_xW2Kz8nS3DLXbrW2zWpTS2MmyVGW41zxmB3z8KdVW3LBKgh4tx2BWW47Dg873by_5VW38kjkD47v83MW43HzXP2HVQ0yW3K3L1w4rF2GsW3_zLV43DYf5ZW4mksjL4pjT9JW3GZj8S2xWGjLW4pqjxG21bpR10\\u0026_hsenc=p2ANqtz--upTI4xtxh18q0Tuk9UdY357sy_UsWGU3g9kT3Mb8mrzvVDgSqHzLd4wdCxC7e5ULgXqp1Z3P7aEcAcp12W5UHWlVp-A\\u0026_hsmi=129513735 )\",\"labels\":[\"CATEGORY_PROMOTIONS\",\"UNREAD\",\"INBOX\"],\"headers\":{\"ARC-Authentication-Results\":\"i=1; mx.google.com;       dkim=pass header.i=@bf01.eu1.hubspotstarter.net header.s=hs1 header.b=NNiaVg+x;       dkim=pass header.i=@eichenwerk.at header.s=hs1-25357905 header.b=LZV9evDi;       spf=pass (google.com: domain of 1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net designates 143.244.84.161 as permitted sender) smtp.mailfrom=\\\"1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net\\\";       dmarc=pass (p=NONE sp=NONE dis=NONE) header.from=eichenwerk.at\",\"ARC-Message-Signature\":\"i=1; a=rsa-sha256; c=relaxed/relaxed; d=google.com; s=arc-20240605;        h=feedback-id:list-unsubscribe-post:precedence:mime-version:subject         :message-id:to:reply-to:from:date:list-unsubscribe:dkim-signature         :dkim-signature;        bh=lez4hoGAC9cA8VzrvkDFldkIbUUvPzE7jkKugidr2e4=;        fh=3jTgowhSmX/Yxq6vZoZ8ZaBHSO1O3L4jxXHHrreJdhw=;        b=T8BdJgFyiN4tCY8buWc+lSjCwL4qmUtkeqne2CTexEewgtsldM9b85q+reQW+uyk39         pAGCtvzt5lvQDSpssJwrPOUk0o8JtppTeDvmXrT7qETeKuGXp9XBEu0hailqBBBm2GVE         TXFw27NhQR92SZ8dA4LuwNb32rGQ9EXA30HQtg4BOXsnXEoZGaHvfX2oe3QnNhkikYrs         DDrK9B1LBJVT0nRVTERhI/J0gzPZ1k25pSxJV6BHrSTkZUJPhgY8VMr56aBhEaDKiZBa         VpJkl8pXavxuSsGY7bR1CRr5k8zxIErbX7ze37XoxSAIGHxhwPGkQRu9wImsrIg2Y6gR         T6ag==;        dara=google.com\",\"ARC-Seal\":\"i=1; a=rsa-sha256; t=1772000574; cv=none;        d=google.com; s=arc-20240605;        b=JGliZITpkdsrGusnwSX5fJItft82f4cJGe+s/d86Qc7IQLGXH7+VvgrrxLsbkQQdCP         CeaXTMf5uUdLj6N5Ir6IZjBGvB1I27xg0oQZuyrwCPrAg86n7Pz6xErQiRXOTr31YO+c         MuIG5cuvRNzcX7hSqhsaDFw7VJG2iQXIHYmCRkfZN9Lm8BPLIFKULQlzpuPHu8E5ks3z         Mr7qvZ0SymtomgiyDJ7q1CLm5Vvj5pGv5miIV4aDoLQdnVwOY8tqhBM60d6IZgOMne6o         OgGU3V1TB/8wwIlqfdtpXxZ4PTsnoOFA0ha7r4fIPWYpiJJvgZcACn/j4gz0vIMD6Imj         7xug==\",\"Authentication-Results\":\"mx.google.com;       dkim=pass header.i=@bf01.eu1.hubspotstarter.net header.s=hs1 header.b=NNiaVg+x;       dkim=pass header.i=@eichenwerk.at header.s=hs1-25357905 header.b=LZV9evDi;       spf=pass (google.com: domain of 1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net designates 143.244.84.161 as permitted sender) smtp.mailfrom=\\\"1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net\\\";       dmarc=pass (p=NONE sp=NONE dis=NONE) header.from=eichenwerk.at\",\"Content-Type\":\"multipart/alternative; boundary=\\\"----=_Part_371339_978480836.1772000574411\\\"\",\"DKIM-Signature\":\"v=1; s=hs1-25357905; d=eichenwerk.at;         i=@eichenwerk.at;         h=sender:from:from:reply-to:to:to:cc:cc:subject:subject:list-unsubscribe:form-sub:feedback-id:list-unsubscribe-post;         a=rsa-sha256; c=relaxed/relaxed;         bh=lez4hoGAC9cA8VzrvkDFldkIbUUvPzE7jkKugidr2e4=;         b=LZV9evDicjJML0GkeaVn/iWfOWUBatMJxrnlDo7RpeIMQugN9SWJH1yoVPKVbp         ZGaVWMjz6Apif8TSlyfSkvl5JcDdqbpimbp3XtHz2FPmn+HmWgsnYEuxG/vZlc/         ff0q6jeXH14bLfCQn3z4Ik0n67I2mMkVfo6WWg/cua3WOIXXHW40zA/+iFy+IAt         jFaYGMm496W1v+UZTYNjuOzBSFnJb8vUEmrxGXxLreNfFD6TUB0/s2frYYHxZuN         JLRWCISUtveT2jlKKd6DX/8UpicbBYKJLg5cq+wxtbWMqDsrPWimiuL78DkS4mg         JLtshh3nLzwvAdTTQjeNVouVIQYw==; q=dns/txt; t=1772000574;         x=1772263374;\",\"Date\":\"Wed, 25 Feb 2026 07:22:54 +0100\",\"Delivered-To\":\"reneheinzl@gmail.com\",\"Feedback-ID\":\"aepdskj:aicfd7p1:aidf:HubSpot\",\"From\":\"EICHENWERK \\u003coffice@eichenwerk.at\\u003e\",\"List-Unsubscribe\":\"\\u003cmailto:1axcumjdiysxvb9n1q0ugfyzsjkihdf2suloww-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net?subject=unsubscribe\\u003e, \\u003chttps://hs-25357905.s.hubspotstarter-eu1.net/subscription-preferences/v2/unsubscribe-all?data=W2nVjwf3Y2x4cW1QnmPS3ZXfGwW1ZkW7l43zkL9W3g6dK849M32kW2zJZ__1L8FY3W3QPxzY2HZYPcW4hs4K149slPgW1ZwX0r3K4ct8W4fpTxg4cPqBKW49BRVY2Wrcn1W3ZDSFf1L9zT-W36FY9_3SPQNrW41C_xp3M7jf3W2zKpZz2CTxbVW4cQGv13h-wqGW41DW523dpr6YW1Qm_Hn2WLWb5W47C2zL2CD9f-W1_b_xY1X6sZHW3_SLhZ4pmJLgW3ZTn-01LGQ6bW4hp_-z1LljHfW45VFw02xKbWNW2zTLm93ZtBBzW4mKdLw1XnBCJW34z-YN3SLW8NW34pN-g2FLjN6W4cHd5t3-0fqDW1NyjQ_2w37-3W1Bs3G_4fjNpCW3DMkf14tG70TW2Hsw7Y1LF_j-W3zb22J258j-QW2YykQm36m_5LW3c-87f21nkVqW1V9X5g1LwT4GW4hm-Jp3KcHkZW2zxtbt41DQCWW3XYL_143RtJbW2358412YmlcSW2HtL8h3_SLjnW3QQRG024ShRpW4rHB982Tgl75W1NnY582WKFTzW2TBpHK4cL78dW4pJFkC36gCmLW3M65bn2KXKBSW1BxJTy3LYZLfW3VHlft3BVY28W2FZmnT47VmjCW4knB4n4mtprCW4pylHY1X9Qc0W4ff0YC4fjsbbW30HSn72HPTt3W3LYZWf3z2WnHf2nD_vW04\\u003e\",\"List-Unsubscribe-Post\":\"List-Unsubscribe=One-Click\",\"MIME-Version\":\"1.0\",\"Message-ID\":\"\\u003c1772000572728.1465bea2-dcf3-4902-a504-de0f75a40f83@bf01.eu1.hubspotstarter.net\\u003e\",\"Precedence\":\"bulk\",\"Received\":\"by 172.16.38.52 with SMTP id a089896hoz74522ffiphsmac28zz1brbr8rb1kykln;        Wed, 25 Feb 2026 06:22:54 GMT\",\"Received-SPF\":\"pass (google.com: domain of 1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net designates 143.244.84.161 as permitted sender) client-ip=143.244.84.161;\",\"Reply-To\":\"office@eichenwerk.at\",\"Return-Path\":\"\\u003c1axbik738umw0tsuqa83bgykly9x2dznev2xq4-reneheinzl=gmail.com@bf01.eu1.hubspotstarter.net\\u003e\",\"Subject\":\"Der Frühling kommt mit neuen Terminen! \",\"To\":\"reneheinzl@gmail.com\",\"X-HS-Cid\":\"1axfanp4mrbz4cmkfutffu15eigxtiailgmstc\",\"X-Received\":\"by 2002:a05:6000:220b:b0:435:a194:734 with SMTP id ffacd0b85a97d-4398fa5ed3fmr1649380f8f.5.1772000574993;        Tue, 24 Feb 20\n... (truncated)",
    "is_error": false
  },
  {
    "name": "drive_search",
    "input": {
      "query": "modifiedTime \u003e= '2026-02-18'"
    },
    "output": "[{\"id\":\"1idVfN0NjQGCLcRwy-WloTGveknYpeIv8-xEH5glRL5I\",\"name\":\"supplement_protocol_GoogleSheet\",\"mime_type\":\"application/vnd.google-apps.spreadsheet\",\"size\":4694,\"parents\":[\"0AD8pn8ilyS-aUk9PVA\"],\"created\":\"2026-02-19T14:34:10.301Z\",\"modified\":\"2026-02-19T14:34:12.333Z\"},{\"id\":\"1zcuQSMmL4x3Hgb7B5L21vx1S5CcOjIG3\",\"name\":\"Projekte\",\"mime_type\":\"application/vnd.google-apps.folder\",\"size\":0,\"parents\":[\"0AD8pn8ilyS-aUk9PVA\"],\"created\":\"2026-02-18T15:30:52.356Z\",\"modified\":\"2026-02-18T15:30:52.356Z\"},{\"id\":\"1cys_MBWfVPa34cmkKuLY9LQkDZpO-V9f\",\"name\":\"Backup-2026\",\"mime_type\":\"application/vnd.google-apps.folder\",\"size\":0,\"parents\":[\"0AD8pn8ilyS-aUk9PVA\"],\"created\":\"2026-02-18T12:00:06.607Z\",\"modified\":\"2026-02-18T12:13:37.227Z\"},{\"id\":\"15-0IcVSP7AtCiZcU9HhxfUJzXIwzKanG\",\"name\":\"Projekte Privat\",\"mime_type\":\"application/vnd.google-apps.folder\",\"size\":0,\"parents\":[\"0AD8pn8ilyS-aUk9PVA\"],\"created\":\"2026-02-18T11:58:27.232Z\",\"modified\":\"2026-02-18T11:58:27.232Z\"},{\"id\":\"1q011XRof_n98VrvBggydQG5uflPfAjY6\",\"name\":\"Familie\",\"mime_type\":\"application/vnd.google-apps.folder\",\"size\":0,\"parents\":[\"0AD8pn8ilyS-aUk9PVA\"],\"created\":\"2026-02-18T11:58:26.644Z\",\"modified\":\"2026-02-18T11:58:26.644Z\"},{\"id\":\"13HmY8jCZ__0AARFws1QqZQtKJ4665bp_\",\"name\":\"Privat\",\"mime_type\":\"application/vnd.google-apps.folder\",\"size\":0,\"parents\":[\"0AD8pn8ilyS-aUk9PVA\"],\"created\":\"2026-02-18T11:58:23.943Z\",\"modified\":\"2026-02-18T11:58:23.943Z\"}]",
    "is_error": false
  }
]
```

**Final Response:**
```
The cloud storage directory contains the following items:

1. **Google Sheet**:  
   - Name: `supplement_protocol_GoogleSheet`  
   - MIME Type: `application/vnd.google-apps.spreadsheet`  
   - Last Modified: 2026-02-19  

2. **Folders**:  
   - **Projekte** (Created: 2026-02-18)  
   - **Backup-2026** (Created: 2026-02-18, Last Modified: 2026-02-18)  
   - **Projekte Privat** (Created: 2026-02-18)  
   - **Familie** (Created: 2026-02-18)  
   - **Privat** (Created: 2026-02-18)  

All items are located in the root folder (`0AD8pn8ilyS-aUk9PVA`). Let me know if you need further details!
```

</details>
