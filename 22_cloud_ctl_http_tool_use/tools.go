// Tutorial 22: Cloud-CTL HTTP Tool Use Benchmark
//
// Demonstrates: llm.Tool, llm.ToolCall, llm.NewAssistantMessage,
//               llm.NewToolResultMessage, agentic tool-use loop,
//               multi-model benchmarking with real-world cloud tools.
//
// This file defines the 6 read-only tools that map to cloud-ctl HTTP REST API
// endpoints and executes them via HTTP GET requests to the cloud-ctl server.
package cloud_ctl_http_tool_use

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitlab2024.bds421-cloud.com/bds421/rho/llm"
)

// =============================================================================
// Tool Definitions
// =============================================================================

// CloudTools returns the 6 read-only tools matching cloud-ctl HTTP API endpoints.
func CloudTools() []llm.Tool {
	return []llm.Tool{
		{
			Name:        "drive_ls",
			Description: "List files and folders in cloud storage. Maps to `GET /api/v1/files`.",
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
			Description: "Search for files in cloud storage using Google Drive API query syntax. Maps to `GET /api/v1/files?q=Q`.",
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
			Description: "Search emails by query. Maps to `GET /api/v1/emails?q=Q`. Supports Gmail-style queries: from:, to:, subject:, is:unread, has:attachment, after:, before:.",
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
			Description: "List calendar events in a date range. Maps to `GET /api/v1/calendar/events`.",
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
			Description: "List all available calendars. Maps to `GET /api/v1/calendar/calendars`.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
		{
			Name:        "sheets_read",
			Description: "Read data from a spreadsheet. Maps to `GET /api/v1/sheets/:id`.",
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
// Tool Executor (live via HTTP REST API)
// =============================================================================

// ExecuteToolHTTP runs a tool call via HTTP GET to the cloud-ctl REST API.
// Requires the cloud-ctl server running at baseURL (e.g. http://localhost:8085).
func ExecuteToolHTTP(baseURL, name string, input any) (string, bool) {
	args, _ := input.(map[string]any)
	if args == nil {
		args = map[string]any{}
	}

	var endpoint string
	params := url.Values{}

	switch name {
	case "drive_ls":
		endpoint = "/api/v1/files"
		if parent, ok := args["parent"].(string); ok && parent != "" {
			params.Set("parent", parent)
		}
		if query, ok := args["query"].(string); ok && query != "" {
			params.Set("q", query)
		}
		params.Set("max", maxOrDefault(args, "max", 10))

	case "drive_search":
		query, _ := args["query"].(string)
		if query == "" {
			return `{"error": "query parameter is required"}`, true
		}
		endpoint = "/api/v1/files"
		params.Set("q", query)
		params.Set("max", maxOrDefault(args, "max", 10))

	case "email_search":
		query, _ := args["query"].(string)
		if query == "" {
			return `{"error": "query parameter is required"}`, true
		}
		endpoint = "/api/v1/emails"
		params.Set("q", query)
		params.Set("max", maxOrDefault(args, "max", 10))

	case "calendar_list":
		endpoint = "/api/v1/calendar/events"
		if calID, ok := args["calendar_id"].(string); ok && calID != "" {
			params.Set("calendar_id", calID)
		}
		if from, ok := args["from"].(string); ok && from != "" {
			params.Set("from", from)
		}
		if to, ok := args["to"].(string); ok && to != "" {
			params.Set("to", to)
		}
		if maxResults := maxOrDefault(args, "max", 20); maxResults != "" {
			params.Set("max_results", maxResults)
		}

	case "calendar_calendars":
		endpoint = "/api/v1/calendar/calendars"

	case "sheets_read":
		id, _ := args["spreadsheet_id"].(string)
		if id == "" {
			return `{"error": "spreadsheet_id parameter is required"}`, true
		}
		endpoint = "/api/v1/sheets/" + url.PathEscape(id)

	default:
		return fmt.Sprintf(`{"error": "unknown tool: %s"}`, name), true
	}

	reqURL := baseURL + endpoint
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		return fmt.Sprintf(`{"error": "HTTP request failed: %s"}`, err.Error()), true
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to read response: %s"}`, err.Error()), true
	}

	result := strings.TrimSpace(string(body))

	// Treat non-2xx status codes as errors.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, true
	}

	// Cap tool output to ~16KB to avoid overwhelming the LLM context.
	const maxOutputBytes = 16384
	if len(result) > maxOutputBytes {
		result = result[:maxOutputBytes] + "\n... (truncated)"
	}
	return result, false
}

// maxOrDefault returns the "max" arg as string, or the default if not set.
func maxOrDefault(args map[string]any, key string, def int) string {
	if v, ok := args[key].(float64); ok && v > 0 {
		return fmt.Sprintf("%d", int(v))
	}
	return fmt.Sprintf("%d", def)
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
