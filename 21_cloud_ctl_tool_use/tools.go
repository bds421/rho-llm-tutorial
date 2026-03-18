// Tutorial 21: Cloud-CTL Tool Use Benchmark
//
// Demonstrates: llm.Tool, llm.ToolCall, llm.NewAssistantMessage,
//               llm.NewToolResultMessage, agentic tool-use loop,
//               multi-model benchmarking with real-world cloud tools.
//
// This file defines the 6 read-only tools that map to cloud-ctl CLI commands
// and returns mock responses (no external dependencies needed).
package cloud_ctl_tool_use

import (
	"fmt"
	"strings"
	"time"

	"github.com/bds421/rho-llm"
)

// =============================================================================
// Tool Definitions
// =============================================================================

// CloudTools returns the 6 read-only tools matching cl subcommands.
func CloudTools() []llm.Tool {
	return []llm.Tool{
		{
			Name:        "drive_ls",
			Description: "List files and folders in cloud storage. Maps to `cl drive ls --json`.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"parent": map[string]any{
						"type":        "string",
						"description": "Parent folder ID to list. Defaults to root.",
					},
					"query": map[string]any{
						"type":        "string",
						"description": "Optional search query to filter results (e.g. file type, name pattern).",
					},
					"max": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results to return. Defaults to 20.",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "drive_search",
			Description: "Search for files in cloud storage using Google Drive API query syntax. Maps to `cl drive search \"Q\" --json`.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Google Drive API query string. Examples: \"name contains 'budget'\", \"mimeType = 'application/pdf'\", \"name contains 'report' and mimeType = 'application/pdf'\".",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "email_search",
			Description: "Search emails by query. Maps to `cl email search \"Q\" --json`. Supports Gmail-style queries: from:, to:, subject:, is:unread, has:attachment, after:, before:.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Email search query (e.g. 'from:alice subject:budget is:unread').",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "calendar_list",
			Description: "List calendar events in a date range. Maps to `cl calendar list --json`.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"from": map[string]any{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format. Defaults to today.",
					},
					"to": map[string]any{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format. Defaults to 7 days from start.",
					},
					"calendar_id": map[string]any{
						"type":        "string",
						"description": "Calendar ID to query. Defaults to primary calendar.",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "calendar_calendars",
			Description: "List all available calendars. Maps to `cl calendar calendars --json`.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
		{
			Name:        "sheets_read",
			Description: "Read data from a spreadsheet. Maps to `cl sheets read ID --json`.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"spreadsheet_id": map[string]any{
						"type":        "string",
						"description": "The spreadsheet ID to read.",
					},
				},
				"required": []string{"spreadsheet_id"},
			},
		},
	}
}

// =============================================================================
// Tool Call Trace (for reporting)
// =============================================================================

// ToolTrace records one tool invocation during an agentic loop.
type ToolTrace struct {
	Name    string `json:"name"`
	Input   any    `json:"input"`
	Output  string `json:"output"`
	IsError bool   `json:"is_error"`
}

// =============================================================================
// Tool Executor (mock responses — no external dependencies)
// =============================================================================

// ExecuteTool returns realistic static JSON for each tool.
// The benchmark tests tool *selection* and *argument construction*; actual API
// data is irrelevant to scoring, so mock responses are sufficient.
func ExecuteTool(name string, input any) (string, bool) {
	args, _ := input.(map[string]any)
	if args == nil {
		args = map[string]any{}
	}

	switch name {
	case "drive_ls":
		return mockDriveLS(), false

	case "drive_search":
		query, _ := args["query"].(string)
		if query == "" {
			return `{"error": "query parameter is required"}`, true
		}
		if strings.Contains(strings.ToLower(query), "xylophonenebulatractor") {
			return `{"files": [], "total": 0}`, false
		}
		return mockDriveSearch(query), false

	case "email_search":
		query, _ := args["query"].(string)
		if query == "" {
			return `{"error": "query parameter is required"}`, true
		}
		return mockEmailSearch(), false

	case "calendar_list":
		return mockCalendarList(), false

	case "calendar_calendars":
		return mockCalendarCalendars(), false

	case "sheets_read":
		id, _ := args["spreadsheet_id"].(string)
		if id == "" {
			return `{"error": "spreadsheet_id parameter is required"}`, true
		}
		if id == "invalid-xyz-999" {
			return `{"error": "spreadsheet not found: invalid-xyz-999 does not exist or you do not have access"}`, true
		}
		return mockSheetsRead(id), false

	default:
		return fmt.Sprintf(`{"error": "unknown tool: %s"}`, name), true
	}
}

// ---------------------------------------------------------------------------
// Mock data generators
// ---------------------------------------------------------------------------

func mockDriveLS() string {
	return `{
  "files": [
    {"id": "1a2b3c", "name": "Q1 Budget.xlsx", "mimeType": "application/vnd.google-apps.spreadsheet", "modifiedTime": "2026-03-17T14:30:00Z", "size": "45678"},
    {"id": "4d5e6f", "name": "Project Plan.docx", "mimeType": "application/vnd.google-apps.document", "modifiedTime": "2026-03-16T09:15:00Z", "size": "23456"},
    {"id": "7g8h9i", "name": "Team Photo.png", "mimeType": "image/png", "modifiedTime": "2026-03-15T11:00:00Z", "size": "1234567"},
    {"id": "j0k1l2", "name": "Meeting Notes.pdf", "mimeType": "application/pdf", "modifiedTime": "2026-03-14T16:45:00Z", "size": "89012"},
    {"id": "m3n4o5", "name": "Design Spec.pdf", "mimeType": "application/pdf", "modifiedTime": "2026-03-13T10:20:00Z", "size": "67890"}
  ],
  "total": 5
}`
}

func mockDriveSearch(query string) string {
	return fmt.Sprintf(`{
  "files": [
    {"id": "s1r2ch", "name": "Q1 Budget Report.pdf", "mimeType": "application/pdf", "modifiedTime": "2026-03-17T14:30:00Z", "size": "78901"},
    {"id": "s3r4ch", "name": "Annual Review.pdf", "mimeType": "application/pdf", "modifiedTime": "2026-03-10T08:00:00Z", "size": "56789"},
    {"id": "s5r6ch", "name": "Spreadsheet Data.xlsx", "mimeType": "application/vnd.google-apps.spreadsheet", "modifiedTime": "2026-03-12T13:45:00Z", "size": "34567"}
  ],
  "query": %q,
  "total": 3
}`, query)
}

func mockEmailSearch() string {
	return `{
  "messages": [
    {"id": "msg001", "threadId": "thr001", "from": "alice@example.com", "to": "me@example.com", "subject": "Q1 Budget Review", "snippet": "Hi, please review the attached budget spreadsheet before our meeting on Thursday.", "date": "2026-03-17T10:00:00Z", "isUnread": true, "hasAttachment": true},
    {"id": "msg002", "threadId": "thr002", "from": "bob@example.com", "to": "me@example.com", "subject": "Project Update", "snippet": "The sprint review is scheduled for Friday. Please prepare your demo.", "date": "2026-03-16T15:30:00Z", "isUnread": true, "hasAttachment": false},
    {"id": "msg003", "threadId": "thr003", "from": "carol@example.com", "to": "me@example.com", "subject": "Design Feedback", "snippet": "I've left comments on the design spec. Can we discuss tomorrow?", "date": "2026-03-15T09:00:00Z", "isUnread": false, "hasAttachment": false}
  ],
  "total": 3
}`
}

func mockCalendarList() string {
	return `{
  "events": [
    {"id": "evt001", "summary": "Team Standup", "start": "2026-03-18T09:00:00Z", "end": "2026-03-18T09:30:00Z", "location": "Zoom", "attendees": ["alice@example.com", "bob@example.com"]},
    {"id": "evt002", "summary": "Q1 Budget Review", "start": "2026-03-19T14:00:00Z", "end": "2026-03-19T15:00:00Z", "location": "Conference Room A", "attendees": ["alice@example.com", "carol@example.com"]},
    {"id": "evt003", "summary": "Sprint Review", "start": "2026-03-21T10:00:00Z", "end": "2026-03-21T11:00:00Z", "location": "Zoom", "attendees": ["bob@example.com", "dave@example.com"]},
    {"id": "evt004", "summary": "1:1 with Manager", "start": "2026-03-20T16:00:00Z", "end": "2026-03-20T16:30:00Z", "location": "", "attendees": ["manager@example.com"]}
  ],
  "total": 4
}`
}

func mockCalendarCalendars() string {
	return `{
  "calendars": [
    {"id": "primary", "summary": "My Calendar", "description": "Primary calendar", "timeZone": "America/New_York", "accessRole": "owner"},
    {"id": "team@example.com", "summary": "Team Calendar", "description": "Shared team events", "timeZone": "America/New_York", "accessRole": "reader"},
    {"id": "holidays@group.v.calendar.google.com", "summary": "US Holidays", "description": "Public holidays", "timeZone": "America/New_York", "accessRole": "reader"}
  ],
  "total": 3
}`
}

func mockSheetsRead(id string) string {
	return fmt.Sprintf(`{
  "spreadsheetId": %q,
  "title": "Q1 Budget",
  "sheets": [
    {
      "name": "Summary",
      "rows": [
        ["Category", "Budget", "Actual", "Variance"],
        ["Engineering", "150000", "142000", "8000"],
        ["Marketing", "80000", "85000", "-5000"],
        ["Operations", "60000", "58000", "2000"],
        ["Total", "290000", "285000", "5000"]
      ]
    }
  ]
}`, id)
}

// SystemPrompt returns the system prompt with today's date for calendar context.
func SystemPrompt() string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`You are a helpful assistant with access to cloud productivity tools.
Today's date is %s.

You have access to the following tools for searching and reading cloud data:
- drive_ls: List files in cloud storage
- drive_search: Search files by name or content
- email_search: Search emails (supports from:, subject:, is:unread, etc.)
- calendar_list: List calendar events in a date range
- calendar_calendars: List available calendars
- sheets_read: Read spreadsheet data by ID

Use the tools to answer the user's questions. Always use tools before answering — do not guess or make up data.
When combining results from multiple tools, synthesize a coherent summary.`, today)
}
