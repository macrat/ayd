package endpoint

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
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
		case "ping", "ping4", "ping6":
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

func (q JQQuery) Run(ctx context.Context, s Store, logScope string, input any) (output MCPOutput) {
	defer func() {
		if r := recover(); r != nil {
			s.ReportInternalError(logScope, fmt.Sprintf("panic occurred: %v", r))
			output = MCPOutput{
				Error: "internal server error",
			}
		}
	}()

	var outputs []any

	iter := q.Code.RunWithContext(ctx, input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if halt, ok := v.(*gojq.HaltError); ok {
			if halt.ExitCode() == 0 {
				break
			}
			v := map[string]any{
				"status":    "halt_error",
				"exit_code": halt.ExitCode(),
				"value":     halt.Value(),
			}
			outputs = append(outputs, v)
			break
		} else if err, ok := v.(error); ok {
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
	Query string `json:"query,omitempty" jsonschema:"A query string to filter status, in jq syntax. Query receives an object like '{\"probe_history\": {\"{target_url}\": {\"status\": \"{status}\", \"updated\": \"{datetime}\", \"records\": [...]}}, \"current_incidents\": [{...}, ...], \"incident_history\": [{...}, ...]}'. You can use 'parse_url' filter to parse target URLs. For example, '.probe_history | to_entries[] | {target: .key, status: .value.status}' to get the current status of all targets."`
}

func FetchStatusByJq(ctx context.Context, s Store, input MCPStatusInput) MCPOutput {
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

	return query.Run(ctx, s, "mcp/query_status", obj)
}

type MCPLogsInput struct {
	Since string `json:"since" jsonschema:"The start time for fetching logs, in RFC3339 format."`
	Until string `json:"until" jsonschema:"The end time for fetching logs, in RFC3339 format."`
	Query string `json:"query,omitempty" jsonschema:"A query string to filter logs, in jq syntax. Query receives an array of status objects. You can use 'parse_url' filter to parse target URLs. For example, 'map(select(.status != \"HEALTHY\")) | group_by(.target)[] | {target: .[0].target, count: length, max_latency: (map(.latency_ms) | max)}' to get unhealthy logs grouped by target with maximum latency.'"`
}

func FetchLogsByJq(ctx context.Context, s Store, input MCPLogsInput) MCPOutput {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if input.Since == "" || input.Until == "" {
		return MCPOutput{
			Error: "since and until parameters are required",
		}
	}

	since, err := api.ParseTime(input.Since)
	if err != nil {
		return MCPOutput{
			Error: fmt.Sprintf("since time must be in RFC3339 format but got %q", input.Since),
		}
	}
	until, err := api.ParseTime(input.Until)
	if err != nil {
		return MCPOutput{
			Error: fmt.Sprintf("until time must be in RFC3339 format but got %q", input.Until),
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

	return query.Run(ctx, s, "mcp/query_logs", records)
}

func MCPServer(s Store) *mcp.Server {
	impl := &mcp.Implementation{
		Name:    "ayd",
		Version: meta.Version,
		Title:   "Ayd",
	}

	opts := &mcp.ServerOptions{
		Instructions: "Ayd is a simple alive monitoring tool. The logs and status can be large, so it is recommended to extract necessary information using jq queries instead of fetching all data at once.",
	}

	if s.Name() != "" {
		impl.Title = fmt.Sprintf("Ayd (%s)", s.Name())
		opts.Instructions = fmt.Sprintf(`%s This Ayd instance's name is %q.`, opts.Instructions, s.Name())
	}

	server := mcp.NewServer(impl, opts)

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
		Description: "Fetch health check logs using jq query from Ayd server. The logs may include extra information than query_status tool.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   true,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MCPLogsInput) (*mcp.CallToolResult, MCPOutput, error) {
		output := FetchLogsByJq(ctx, s, input)
		return nil, output, nil
	})

	return server
}

func MCPHandler(s Store) http.Handler {
	server := MCPServer(s)

	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})

	return handler
}
