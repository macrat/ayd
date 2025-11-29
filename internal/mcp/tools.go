package mcp

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/macrat/ayd/internal/query"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StatusInput is the input for query_status tool.
type StatusInput struct {
	JQ string `json:"jq,omitempty" jsonschema:"A jq query string to filter and/or aggregate status. Query receives an array. Each object is like '{\"target\": \"{url}\", \"status\": \"...\", \"latest_log\": {\"time\": \"{RFC 3339}\", \"status\": \"...\", \"latency\": ..., \"message\": \"...\", ...}}'. You can use 'parse_url' filter to parse target URLs. For example, '.[] | {target: .target, status: .status, message: .latest_log.message}' to get the current status of all targets."`
}

// FetchStatusByJQ fetches status from store and applies jq query.
func FetchStatusByJQ(ctx context.Context, s Store, input StatusInput) (Output, error) {
	jq, err := ParseJQ(input.JQ)
	if err != nil {
		return Output{}, fmt.Errorf("failed to parse jq query: %w", err)
	}

	history := s.ProbeHistory()

	targets := make([]any, 0, len(history))

	for _, r := range history {
		var latest map[string]any
		if len(r.Records) > 0 {
			latest = RecordToMap(r.Records[len(r.Records)-1])
			delete(latest, "target")
		}

		targets = append(targets, map[string]any{
			"target":     r.Target.String(),
			"status":     r.Status.String(),
			"latest_log": latest,
		})
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].(map[string]any)["target"].(string) < targets[j].(map[string]any)["target"].(string)
	})

	return jq.Run(ctx, targets)
}

// IncidentsInput is the input for query_incidents tool.
type IncidentsInput struct {
	IncludeOngoing  *bool  `json:"include_ongoing,omitempty" jsonschema:"Whether to include ongoing incidents in the result. If omitted, ongoing incidents are included."`
	IncludeResolved bool   `json:"include_resolved,omitempty" jsonschema:"Whether to include resolved incidents in the result. If omitted, resolved incidents are not included."`
	JQ              string `json:"jq,omitempty" jsonschema:"A jq query string to filter and/or aggregate incidents. Query receives an array. Each object is like '{\"target\": \"{url}\", \"status\": \"...\", \"message\": \"...\", \"starts_at\": \"{RFC 3339}\", \"ends_at\": \"{RFC 3339 or null}\"}'. You can use 'parse_url' filter to parse target URLs. For example, 'map(.target | startswith(\"http\"))[] | {target: .target, status: .status, starts_at: .starts_at, resolved: (.ends_at != null)}' to get incidents of HTTP/HTTPS targets."`
}

// FetchIncidentsByJQ fetches incidents from store and applies jq query.
func FetchIncidentsByJQ(ctx context.Context, s Store, input IncidentsInput) (Output, error) {
	jq, err := ParseJQ(input.JQ)
	if err != nil {
		return Output{}, fmt.Errorf("failed to parse jq query: %w", err)
	}

	current := s.CurrentIncidents()
	history := s.IncidentHistory()

	count := 0
	if input.IncludeResolved {
		count += len(history)
	}
	if input.IncludeOngoing == nil || *input.IncludeOngoing {
		count += len(current)
	}

	incidents := make([]any, 0, count)

	if input.IncludeResolved {
		for _, v := range history {
			incidents = append(incidents, IncidentToMap(v))
		}
	}

	if input.IncludeOngoing == nil || *input.IncludeOngoing {
		for _, v := range current {
			incidents = append(incidents, IncidentToMap(v))
		}
	}

	sort.Slice(incidents, func(i, j int) bool {
		return incidents[i].(map[string]any)["starts_at_unix"].(int64) < incidents[j].(map[string]any)["starts_at_unix"].(int64)
	})

	return jq.Run(ctx, incidents)
}

// LogsInput is the input for query_logs tool.
type LogsInput struct {
	Since  string `json:"since" jsonschema:"The start time for fetching logs, in RFC3339 format."`
	Until  string `json:"until" jsonschema:"The end time for fetching logs, in RFC3339 format."`
	Search string `json:"search,omitempty" jsonschema:"A search query to filter logs. For example, 'status!=HEALTHY', or 'status=FAILURE AND (latency<100ms OR target=http://example.com*)'. It is recommended to use this parameter to reduce the number of logs before applying jq query. If omitted, no filtering is applied."`
	JQ     string `json:"jq,omitempty" jsonschema:"A jq query string to filter logs. Query receives an array of status objects. Each objects has at least 'time', 'target', 'status', and 'latency'. You can use 'parse_url' filter to parse target URLs. For example, 'map(select(.status != \"HEALTHY\")) | group_by(.target)[] | {target: .[0].target, count: length, max_latency: (map(.latency_ms) | max)}' to get unhealthy logs grouped by target with maximum latency.'"`
}

// FetchLogsByJQ fetches logs from store and applies jq query.
func FetchLogsByJQ(ctx context.Context, s Store, input LogsInput) (Output, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if input.Since == "" || input.Until == "" {
		return Output{}, errors.New("since and until parameters are required")
	}

	since, err := api.ParseTime(input.Since)
	if err != nil {
		return Output{}, fmt.Errorf("since time must be in RFC3339 format but got %q", input.Since)
	}
	until, err := api.ParseTime(input.Until)
	if err != nil {
		return Output{}, fmt.Errorf("until time must be in RFC3339 format but got %q", input.Until)
	}

	logs, err := s.OpenLog(since, until)
	if err != nil {
		s.ReportInternalError("mcp/query_logs", fmt.Sprintf("failed to open logs: %v", err))
		return Output{}, errors.New("internal server error")
	}
	defer logs.Close()

	var scanner api.LogScanner = logs
	if input.Search != "" {
		var newSince, newUntil *time.Time
		scanner, newSince, newUntil = query.Filter(logs, input.Search)
		if newSince != nil && newSince.After(since) {
			since = *newSince
		}
		if newUntil != nil && newUntil.Before(until) {
			until = *newUntil
		}
	}

	jq, err := ParseJQ(input.JQ)
	if err != nil {
		return Output{}, fmt.Errorf("failed to parse jq query: %w", err)
	}

	records := []any{}
	for scanner.Scan() {
		rec := scanner.Record()
		records = append(records, RecordToMap(rec))
	}

	return jq.Run(ctx, records)
}

// AddReadOnlyTools adds the read-only query tools to the MCP server.
// These tools are: query_status, query_incidents, query_logs.
func AddReadOnlyTools(server *mcp.Server, s Store) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_status",
		Title:       "Query status",
		Description: "Fetch latest status of each targets from Ayd server.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   true,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, Output, error) {
		output, err := FetchStatusByJQ(ctx, s, input)
		return nil, output, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_incidents",
		Title:       "Query incidents",
		Description: "Fetch current and past incidents from Ayd server. The result is limited by number. Please use query_logs tool to analyze long-term history.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   true,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input IncidentsInput) (*mcp.CallToolResult, Output, error) {
		output, err := FetchIncidentsByJQ(ctx, s, input)
		return nil, output, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_logs",
		Title:       "Query logs",
		Description: "Fetch health check logs from Ayd server. The result can be very large. Please use time range and aggregation in jq query to reduce the result size.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   true,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input LogsInput) (*mcp.CallToolResult, Output, error) {
		output, err := FetchLogsByJQ(ctx, s, input)
		return nil, output, err
	})
}
