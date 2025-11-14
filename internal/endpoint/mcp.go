package endpoint

import (
	"fmt"
	"time"
	"net/http"
	"context"
	"maps"
	"strings"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/itchyny/gojq"
)

func recordToMap(rec api.Record) map[string]any {
	x := map[string]any{
		"time":       rec.Time.Format(time.RFC3339),
		"time_unix":  rec.Time.Unix(),
		"status":     rec.Status.String(),
		"latency":    rec.Latency.String(),
		"latency_ms": rec.Latency.Milliseconds(),
		"target":     rec.Target.String(),
		"message":    rec.Message,
	}
	maps.Copy(x, rec.Extra)
	return x
}

func incidentToMap(inc api.Incident) map[string]any {
	return map[string]any{
		"target":    inc.Target.String(),
		"status":    inc.Status.String(),
		"message":   inc.Message,
		"starts_at": inc.StartsAt.Format(time.RFC3339),
		"starts_at_unix": inc.StartsAt.Unix(),
		"ends_at":   inc.EndsAt.Format(time.RFC3339),
		"ends_at_unix": inc.EndsAt.Unix(),
	}
}

type MCPTargetsInput struct {
	Keywords []string `json:"keywords,omitempty" jsonschema:"A list of keywords to filter targets."`
}

type MCPTargetsOutput struct {
	Targets []string `json:"targets" jsonschema:"A list of target URLs includes the keywords."`
}

func FetchTargets(ctx context.Context, s Store, input MCPTargetsInput) (output MCPTargetsOutput) {
	targets := s.Targets()

	filtered := make([]string, 0, len(targets))
	for _, t := range targets {
		matched := true
		for _, kw := range input.Keywords {
			if !strings.Contains(t, kw) {
				matched = false
				break
			}
		}
		if matched {
			filtered = append(filtered, t)
		}
	}

	output.Targets = filtered
	return output
}

type MCPStatusInput struct {
	Query string `json:"query,omitempty" jsonschema:"A query string to filter status, in jq syntax."`
}

type MCPStatusOutput struct {
	Result []any `json:"result" jsonschema:"The result of the status query."`
	Error  string `json:"error,omitempty" jsonschema:"Error message if the query failed."`
}

func FetchStatusByJq(ctx context.Context, s Store, input MCPStatusInput) (output MCPStatusOutput) {
	output.Result = []any{}  // 空の配列で初期化（nilではなく）

	defer func() {
		if r := recover(); r != nil {
			s.ReportInternalError("mcp/query_status", fmt.Sprintf("panic occurred: %v", r))
			output = MCPStatusOutput{
				Result: []any{},  // エラー時も空の配列を返す
				Error:  "internal server error",
			}
		}
	}()

	if input.Query == "" {
		input.Query = "."
	}

	query, err := gojq.Parse(input.Query)
	if err != nil {
		return MCPStatusOutput{
			Result: []any{},
			Error:  fmt.Sprintf("failed to parse query: %v", err),
		}
	}

	report := s.MakeReport(40)

	obj := map[string]any{
		"reported_at": report.ReportedAt.Format(time.RFC3339),
	}

	obj["probe_history"] = map[string]any{}
	for k, v := range report.ProbeHistory {
		h := map[string]any{
			"target":    v.Target.String(),
			"status":    v.Status.String(),
			"updated":   v.Updated.Format(time.RFC3339),
			"records":   make([]any, len(v.Records)),
		}
		for i, r := range v.Records {
			h["records"].([]any)[i] = recordToMap(r)
		}
		obj["probe_history"].(map[string]any)[k] = h
	}

	obj["current_incidents"] = make([]any, len(report.CurrentIncidents))
	for i, v := range report.CurrentIncidents {
		obj["current_incidents"].([]any)[i] = incidentToMap(v)
	}

	obj["incident_history"] = make([]any, len(report.IncidentHistory))
	for i, v := range report.IncidentHistory {
		obj["incident_history"].([]any)[i] = incidentToMap(v)
	}

	iter := query.RunWithContext(ctx, obj)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return MCPStatusOutput{
				Result: []any{},
				Error:  err.Error(),
			}
		}
		output.Result = append(output.Result, v)
	}

	if len(output.Result) == 1 {
		if arr, ok := output.Result[0].([]any); ok && arr != nil {
			output.Result = arr
		}
	}

	return output
}

type MCPLogsInput struct {
	Since string `json:"since" jsonschema:"The start time for fetching logs, in RFC3339 format."`
	Until string `json:"until" jsonschema:"The end time for fetching logs, in RFC3339 format."`
	Query string `json:"query,omitempty" jsonschema:"A query string to filter logs, in jq syntax."`
}

type MCPLogsOutput struct {
	Result []any `json:"result" jsonschema:"The result of the log query."`
	Error  string `json:"error,omitempty" jsonschema:"Error message if the query failed."`
}

func FetchLogsByJq(ctx context.Context, s Store, input MCPLogsInput) (output MCPLogsOutput) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	output.Result = []any{}  // 空の配列で初期化（nilではなく）

	defer func() {
		if r := recover(); r != nil {
			s.ReportInternalError("mcp/query_logs", fmt.Sprintf("panic occurred: %v", r))
			output = MCPLogsOutput{
				Result: []any{},  // エラー時も空の配列を返す
				Error:  "internal server error",
			}
		}
	}()

	since, err := api.ParseTime(input.Since)
	if err != nil {
		return MCPLogsOutput{
			Result: []any{},
			Error:  fmt.Sprintf("invalid since time: %v", err),
		}
	}
	until, err := api.ParseTime(input.Until)
	if err != nil {
		return MCPLogsOutput{
			Result: []any{},
			Error:  fmt.Sprintf("invalid until time: %v", err),
		}
	}

	logs, err := s.OpenLog(since, until)
	if err != nil {
		s.ReportInternalError("mcp", fmt.Sprintf("failed to open logs: %v", err))
		return MCPLogsOutput{
			Result: []any{},
			Error:  "internal server error",
		}
	}
	defer logs.Close()

	if input.Query == "" {
		input.Query = "."
	}

	query, err := gojq.Parse(input.Query)
	if err != nil {
		return MCPLogsOutput{
			Result: []any{},
			Error:  fmt.Sprintf("failed to parse query: %v", err),
		}
	}

	records := []any{}
	for logs.Scan() {
		rec := logs.Record()
		records = append(records, recordToMap(rec))
	}

	iter := query.RunWithContext(ctx, records)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return MCPLogsOutput{
				Result: []any{},
				Error:  err.Error(),
			}
		}
		output.Result = append(output.Result, v)
	}

	if len(output.Result) == 1 {
		if arr, ok := output.Result[0].([]any); ok && arr != nil {
			output.Result = arr
		}
	}

	return output
}

func MCPHandler(s Store) http.HandlerFunc {
	server := mcp.NewServer(&mcp.Implementation{
		Name: "Ayd",
		Version: "0.1.0",  // TODO: set real version
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_targets",
		Description: "List monitored target URLs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MCPTargetsInput) (*mcp.CallToolResult, MCPTargetsOutput, error) {
		output := FetchTargets(ctx, s, input)
		return nil, output, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "query_status",
		Description: "Fetch current status using jq query from Ayd server.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MCPStatusInput) (*mcp.CallToolResult, MCPStatusOutput, error) {
		output := FetchStatusByJq(ctx, s, input)
		return nil, output, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "query_logs",
		Description: "Fetch health check logs using jq query from Ayd server.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MCPLogsInput) (*mcp.CallToolResult, MCPLogsOutput, error) {
		output := FetchLogsByJq(ctx, s, input)
		return nil, output, nil
	})

	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	return handler.ServeHTTP
}
