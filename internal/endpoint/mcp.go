package endpoint

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"github.com/macrat/ayd/internal/meta"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func recordToMap(rec api.Record) map[string]any {
	x := map[string]any{
		"time":       rec.Time.Format(time.RFC3339),
		"time_unix":  rec.Time.Unix(),
		"status":     rec.Status.String(),
		"latency":    rec.Latency.String(),
		"latency_ms": float64(rec.Latency.Nanoseconds()) / 1000000.0,
		"target":     rec.Target.String(),
		"message":    rec.Message,
	}
	maps.Copy(x, rec.Extra)
	return x
}

func incidentToMap(inc api.Incident) map[string]any {
	r := map[string]any{
		"target":         inc.Target.String(),
		"status":         inc.Status.String(),
		"message":        inc.Message,
		"starts_at":      inc.StartsAt.Format(time.RFC3339),
		"starts_at_unix": inc.StartsAt.Unix(),
	}

	if inc.EndsAt.IsZero() {
		r["ends_at"] = nil
		r["ends_at_unix"] = nil
	} else {
		r["ends_at"] = inc.EndsAt.Format(time.RFC3339)
		r["ends_at_unix"] = inc.EndsAt.Unix()
	}

	return r
}

type MCPTargetsInput struct {
	Keywords []string `json:"keywords,omitempty" jsonschema:"A list of keywords to filter targets. They work as an AND condition."`
}

type MCPTargetsOutput struct {
	Targets []string `json:"targets" jsonschema:"A list of target URLs that include the keywords."`
}

func FetchTargets(s Store, input MCPTargetsInput) (output MCPTargetsOutput) {
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

type MCPOutput struct {
	Result any    `json:"result" jsonschema:"The result of the query."`
	Error  string `json:"error,omitempty" jsonschema:"Error message if the query failed."`
}

func jqParseURL(x any, _ []any) any {
	str, ok := x.(string)
	if !ok {
		return fmt.Errorf("parse_url/0: expected a string but got %T (%v)", x, x)
	}
	u, err := url.Parse(str)
	if err != nil {
		return fmt.Errorf("parse_url/0: failed to parse URL: %v", err)
	}

	username := ""
	if u.User != nil {
		username = u.User.Username()
	}

	queries := map[string][]any{}
	for key, vals := range u.Query() {
		queries[key] = make([]any, len(vals))
		for i, v := range vals {
			queries[key][i] = v
		}
	}

	if u.Opaque != "" && u.Host == "" {
		switch u.Scheme {
		case "ping":
			u.Host = u.Opaque
			u.Opaque = ""
		case "dns", "dns4", "dns6", "file", "exec", "mailto", "source":
			u.Path = u.Opaque
			u.Opaque = ""
		}
	}

	return map[string]any{
		"scheme":   u.Scheme,
		"username": username,
		"hostname": u.Hostname(),
		"port":     u.Port(),
		"path":     u.Path,
		"queries":  queries,
		"fragment": u.Fragment,
		"opaque":   u.Opaque,
	}
}

type JQQuery struct {
	Code *gojq.Code
}

func ParseJQ(query string) (JQQuery, error) {
	if query == "" {
		query = "."
	}

	q, err := gojq.Parse(query)
	if err != nil {
		return JQQuery{}, err
	}

	c, err := gojq.Compile(
		q,
		gojq.WithFunction("parse_url", 0, 0, jqParseURL),
	)
	if err != nil {
		return JQQuery{}, err
	}

	return JQQuery{Code: c}, nil
}

func (q JQQuery) Run(ctx context.Context, v any) MCPOutput {
	var outputs []any

	iter := q.Code.RunWithContext(ctx, v)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return MCPOutput{
				Error: err.Error(),
			}
		}
		outputs = append(outputs, v)
	}

	if len(outputs) == 1 {
		return MCPOutput{
			Result: outputs[0],
		}
	} else {
		return MCPOutput{
			Result: outputs,
		}
	}
}

type MCPStatusInput struct {
	Query string `json:"query,omitempty" jsonschema:"A query string to filter status, in jq syntax. Query receives an object like '{\"probe_history\": {\"{target_url}\": {\"status\": \"{status}\", \"updated\": \"{datetime}\", \"records\": [...]}}, \"current_incidents\": [{...}, ...], \"incident_history\": [{...}, ...]}'. You can use 'parse_url' filter to parse target URLs."`
}

func FetchStatusByJq(ctx context.Context, s Store, input MCPStatusInput) (output MCPOutput) {
	defer func() {
		if r := recover(); r != nil {
			s.ReportInternalError("mcp/query_status", fmt.Sprintf("panic occurred: %v", r))
			output = MCPOutput{
				Error: "internal server error",
			}
		}
	}()

	query, err := ParseJQ(input.Query)
	if err != nil {
		return MCPOutput{
			Error: fmt.Sprintf("failed to parse query: %v", err),
		}
	}

	report := s.MakeReport(40)

	obj := map[string]any{}

	obj["probe_history"] = map[string]any{}
	for k, v := range report.ProbeHistory {
		var updated *string
		if !v.Updated.IsZero() {
			u := v.Updated.Format(time.RFC3339)
			updated = &u
		}

		h := map[string]any{
			"status":  v.Status.String(),
			"updated": updated,
			"records": make([]any, len(v.Records)),
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

	return query.Run(ctx, obj)
}

type MCPLogsInput struct {
	Since string `json:"since" jsonschema:"The start time for fetching logs, in RFC3339 format."`
	Until string `json:"until" jsonschema:"The end time for fetching logs, in RFC3339 format."`
	Query string `json:"query,omitempty" jsonschema:"A query string to filter logs, in jq syntax. Query receives an array of status objects. Please try '.[0]' to understand the structure if needed. You can use 'parse_url' filter to parse target URLs."`
}

func FetchLogsByJq(ctx context.Context, s Store, input MCPLogsInput) (output MCPOutput) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			s.ReportInternalError("mcp/query_logs", fmt.Sprintf("panic occurred: %v", r))
			output = MCPOutput{
				Error: "internal server error",
			}
		}
	}()

	if input.Since == "" || input.Until == "" {
		return MCPOutput{
			Error: "since and until parameters are required",
		}
	}

	since, err := api.ParseTime(input.Since)
	if err != nil {
		if errors.Is(err, api.ErrInvalidTime) {
			return MCPOutput{
				Error: fmt.Sprintf("since time must be in RFC3339 format but got %q", input.Since),
			}
		} else {
			return MCPOutput{
				Error: fmt.Sprintf("invalid since time: %v", err),
			}
		}
	}
	until, err := api.ParseTime(input.Until)
	if err != nil {
		if errors.Is(err, api.ErrInvalidTime) {
			return MCPOutput{
				Error: fmt.Sprintf("until time must be in RFC3339 format but got %q", input.Until),
			}
		} else {
			return MCPOutput{
				Error: fmt.Sprintf("invalid until time: %v", err),
			}
		}
	}

	logs, err := s.OpenLog(since, until)
	if err != nil {
		s.ReportInternalError("mcp/query_logs", fmt.Sprintf("failed to open logs: %v", err))
		return MCPOutput{
			Error: "internal server error",
		}
	}
	defer logs.Close()

	query, err := ParseJQ(input.Query)
	if err != nil {
		return MCPOutput{
			Error: fmt.Sprintf("failed to parse query: %v", err),
		}
	}

	records := []any{}
	for logs.Scan() {
		rec := logs.Record()
		records = append(records, recordToMap(rec))
	}

	return query.Run(ctx, records)
}

func MCPHandler(s Store) http.HandlerFunc {
	implName := "Ayd - alive monitoring"
	implTitle := "Ayd"

	if meta.InstanceName != "" {
		implName = fmt.Sprintf("Ayd - alive monitoring - %s", meta.InstanceName)
		implTitle = fmt.Sprintf("Ayd - %s", meta.InstanceName)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    implName,
		Version: meta.Version,
		Title:   implTitle,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_targets",
		Title:       "List targets",
		Description: "List currently monitored target URLs.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   true,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MCPTargetsInput) (*mcp.CallToolResult, MCPTargetsOutput, error) {
		output := FetchTargets(s, input)
		return nil, output, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_status",
		Title:       "Query status",
		Description: "Fetch current status summary using jq query from Ayd server.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   true,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MCPStatusInput) (*mcp.CallToolResult, MCPOutput, error) {
		output := FetchStatusByJq(ctx, s, input)
		return nil, output, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_logs",
		Title:       "Query logs",
		Description: "Fetch health check logs using jq query from Ayd server.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   true,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MCPLogsInput) (*mcp.CallToolResult, MCPOutput, error) {
		output := FetchLogsByJq(ctx, s, input)
		return nil, output, nil
	})

	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})

	return handler.ServeHTTP
}
