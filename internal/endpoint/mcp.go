package endpoint

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/itchyny/gojq"
	"github.com/macrat/ayd/internal/meta"
	"github.com/macrat/ayd/internal/query"
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

func incidentToMap(inc *api.Incident) map[string]any {
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
	Result any `json:"result" jsonschema:"The result of the query."`
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

func (q JQQuery) Run(ctx context.Context, s Store, logScope string, input any) (MCPOutput, error) {
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
			return MCPOutput{}, err
		}
		outputs = append(outputs, v)
	}

	if len(outputs) == 1 {
		return MCPOutput{
			Result: outputs[0],
		}, nil
	} else {
		return MCPOutput{
			Result: outputs,
		}, nil
	}
}

type MCPStatusInput struct {
	JQ string `json:"jq,omitempty" jsonschema:"A jq query string to filter and/or aggregate status. Query receives an array. Each object is like '{\"target\": \"{url}\", \"status\": \"...\", \"latest_log\": {\"time\": \"{RFC 3339}\", \"status\": \"...\", \"latency\": ..., \"message\": \"...\", ...}}'. You can use 'parse_url' filter to parse target URLs. For example, '.[] | {target: .target, status: .status, message: .latest_log.message}' to get the current status of all targets."`
}

func FetchStatusByJq(ctx context.Context, s Store, input MCPStatusInput) (MCPOutput, error) {
	query, err := ParseJQ(input.JQ)
	if err != nil {
		return MCPOutput{}, fmt.Errorf("failed to parse jq query: %w", err)
	}

	history := s.ProbeHistory()

	targets := make([]any, 0, len(history))

	for _, r := range history {
		var latest map[string]any
		if len(r.Records) > 0 {
			latest = recordToMap(r.Records[len(r.Records)-1])
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

	return query.Run(ctx, s, "mcp/query_status", targets)
}

type MCPIncidentsInput struct {
	IncludeOngoing  *bool  `json:"include_ongoing,omitempty" jsonschema:"Whether to include ongoing incidents in the result. If omitted, ongoing incidents are included."`
	IncludeResolved bool   `json:"include_resolved,omitempty" jsonschema:"Whether to include resolved incidents in the result. If omitted, resolved incidents are not included."`
	JQ              string `json:"jq,omitempty" jsonschema:"A jq query string to filter and/or aggregate incidents. Query receives an array. Each object is like '{\"target\": \"{url}\", \"status\": \"...\", \"message\": \"...\", \"starts_at\": \"{RFC 3339}\", \"ends_at\": \"{RFC 3339 or null}\"}'. You can use 'parse_url' filter to parse target URLs. For example, 'map(.target | startswith(\"http\"))[] | {target: .target, status: .status, starts_at: .starts_at, resolved: (.ends_at != null)}' to get incidents of HTTP/HTTPS targets."`
}

func FetchIncidentsByJq(ctx context.Context, s Store, input MCPIncidentsInput) (MCPOutput, error) {
	query, err := ParseJQ(input.JQ)
	if err != nil {
		return MCPOutput{}, fmt.Errorf("failed to parse jq query: %w", err)
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
			incidents = append(incidents, incidentToMap(v))
		}
	}

	if input.IncludeOngoing == nil || *input.IncludeOngoing {
		for _, v := range current {
			incidents = append(incidents, incidentToMap(v))
		}
	}

	sort.Slice(incidents, func(i, j int) bool {
		return incidents[i].(map[string]any)["starts_at_unix"].(int64) < incidents[j].(map[string]any)["starts_at_unix"].(int64)
	})

	return query.Run(ctx, s, "mcp/query_incidents", incidents)
}

type MCPLogsInput struct {
	Since  string `json:"since" jsonschema:"The start time for fetching logs, in RFC3339 format."`
	Until  string `json:"until" jsonschema:"The end time for fetching logs, in RFC3339 format."`
	Search string `json:"search,omitempty" jsonschema:"A search query to filter logs. For example, 'status!=HEALTHY', or 'status=FAILURE AND (latency<100ms OR target=http://example.com*)'. It is recommended to use this parameter to reduce the number of logs before applying jq query. If omitted, no filtering is applied."`
	JQ     string `json:"jq,omitempty" jsonschema:"A jq query string to filter logs. Query receives an array of status objects. Each objects has at least 'time', 'target', 'status', and 'latency'. You can use 'parse_url' filter to parse target URLs. For example, 'map(select(.status != \"HEALTHY\")) | group_by(.target)[] | {target: .[0].target, count: length, max_latency: (map(.latency_ms) | max)}' to get unhealthy logs grouped by target with maximum latency.'"`
}

func FetchLogsByJq(ctx context.Context, s Store, input MCPLogsInput) (MCPOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if input.Since == "" || input.Until == "" {
		return MCPOutput{}, errors.New("since and until parameters are required")
	}

	since, err := api.ParseTime(input.Since)
	if err != nil {
		return MCPOutput{}, fmt.Errorf("since time must be in RFC3339 format but got %q", input.Since)
	}
	until, err := api.ParseTime(input.Until)
	if err != nil {
		return MCPOutput{}, fmt.Errorf("until time must be in RFC3339 format but got %q", input.Until)
	}

	var q query.Query
	if input.Search != "" {
		q = query.ParseQuery(input.Search)
		st, en := q.TimeRange()

		if st != nil && st.After(since) {
			since = *st
		}
		if en != nil && en.Before(until) {
			until = *en
		}
	}

	logs, err := s.OpenLog(since, until)
	if err != nil {
		s.ReportInternalError("mcp/query_logs", fmt.Sprintf("failed to open logs: %v", err))
		return MCPOutput{}, errors.New("internal server error")
	}
	defer logs.Close()

	if q != nil {
		logs = FilterScanner{
			Scanner: logs,
			Query:   q,
		}
	}

	jq, err := ParseJQ(input.JQ)
	if err != nil {
		return MCPOutput{}, fmt.Errorf("failed to parse jq query: %w", err)
	}

	records := []any{}
	for logs.Scan() {
		rec := logs.Record()
		records = append(records, recordToMap(rec))
	}

	return jq.Run(ctx, s, "mcp/query_logs", records)
}

func MCPServer(s Store) *mcp.Server {
	impl := &mcp.Implementation{
		Name:    "ayd",
		Version: meta.Version,
		Title:   "Ayd",
	}

	opts := &mcp.ServerOptions{
		Instructions: "Ayd is a simple alive monitoring tool. The logs and status can be large, so it is recommended to extract necessary information using search and jq queries instead of fetching all data at once.",
	}

	if s.Name() != "" {
		impl.Title = fmt.Sprintf("Ayd (%s)", s.Name())
		opts.Instructions = fmt.Sprintf(`%s This Ayd instance's name is %q.`, opts.Instructions, s.Name())
	}

	server := mcp.NewServer(impl, opts)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_status",
		Title:       "Query status",
		Description: "Fetch latest status of each targets from Ayd server.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   true,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MCPStatusInput) (*mcp.CallToolResult, MCPOutput, error) {
		output, err := FetchStatusByJq(ctx, s, input)
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
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MCPIncidentsInput) (*mcp.CallToolResult, MCPOutput, error) {
		output, err := FetchIncidentsByJq(ctx, s, input)
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
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MCPLogsInput) (*mcp.CallToolResult, MCPOutput, error) {
		output, err := FetchLogsByJq(ctx, s, input)
		return nil, output, err
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
