package endpoint_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/testutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestJQQuery(t *testing.T) {
	input := map[string]any{
		"foo": 1,
		"bar": 2,
		"baz": map[string]any{
			"qux": 3,
		},
	}

	tests := []struct {
		Query  string
		Expect any
		Error  string
	}{
		{
			Query:  `.foo + .bar`,
			Expect: 3,
		},
		{
			Query:  `.baz.qux * 2`,
			Expect: 6,
		},
		{
			Query:  `[.foo, .bar]`,
			Expect: []any{1, 2},
		},
		{
			Query: `0 / 0`,
			Error: `cannot divide number (0) by: number (0)`,
		},
		{
			Query: `"http://foo:bar@example.com/path?query=value#fragment" | parse_url`,
			Expect: map[string]any{
				"scheme":   "http",
				"username": "foo",
				"hostname": "example.com",
				"port":     "",
				"path":     "/path",
				"queries":  map[string][]any{"query": {"value"}},
				"fragment": "fragment",
				"opaque":   "",
			},
		},
		{
			Query: `"ping:example.com" | parse_url`,
			Expect: map[string]any{
				"scheme":   "ping",
				"username": "",
				"hostname": "example.com",
				"port":     "",
				"path":     "",
				"queries":  map[string][]any{},
				"fragment": "",
				"opaque":   "",
			},
		},
		{
			Query: `"dns:example.com" | parse_url`,
			Expect: map[string]any{
				"scheme":   "dns",
				"username": "",
				"hostname": "",
				"port":     "",
				"path":     "example.com",
				"queries":  map[string][]any{},
				"fragment": "",
				"opaque":   "",
			},
		},
		{
			Query: `"://hoge" | parse_url`,
			Error: `parse_url/0: failed to parse URL: parse "://hoge": missing protocol scheme`,
		},
		{
			Query: `123 | parse_url`,
			Error: `parse_url/0: expected a string but got int (123)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Query, func(t *testing.T) {
			jq, err := endpoint.ParseJQ(tt.Query)
			if err != nil {
				t.Fatalf("failed to parse JQ query: %v", err)
			}

			output := jq.Run(context.Background(), input)

			if output.Error != tt.Error {
				if tt.Error == "" {
					t.Fatalf("unexpected error: %v", output.Error)
				} else {
					t.Fatalf("expected error %q, got %q", tt.Error, output.Error)
				}
			} else {
				if diff := cmp.Diff(tt.Expect, output.Result); diff != "" {
					t.Errorf("output mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

type MCPTest[I, O any] struct {
	Name   string
	Args   I
	Expect O
}

func RunMCPTest[I, O any](t *testing.T, tool string, tests []MCPTest[I, O]) {
	srv := testutil.StartTestServer(t)
	t.Cleanup(func() {
		srv.Close()
	})

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tool, tt.Name), func(t *testing.T) {
			client := mcp.NewClient(&mcp.Implementation{
				Name:    "test-client",
				Version: "none",
			}, nil)
			sess, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
				Endpoint: srv.URL + "/mcp",
			}, nil)
			if err != nil {
				t.Fatalf("failed to connect to MCP server: %v", err)
			}
			defer sess.Close()

			result, err := sess.CallTool(t.Context(), &mcp.CallToolParams{
				Name:      tool,
				Arguments: tt.Args,
			})
			if err != nil {
				t.Fatalf("failed to call tool %q: %v", tool, err)
			}

			if len(result.Content) != 1 {
				t.Fatalf("expected 1 content, got %#v", result.Content)
			}

			var resultData O
			if text, ok := result.Content[0].(*mcp.TextContent); !ok {
				t.Fatalf("expected TextContent, got %#v", result.Content[0])
			} else if err := json.Unmarshal([]byte(text.Text), &resultData); err != nil {
				t.Fatalf("failed to unmarshal result data: %v", err)
			}

			if diff := cmp.Diff(tt.Expect, resultData); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMCPHandler_ListTargets(t *testing.T) {
	tests := []MCPTest[endpoint.MCPTargetsInput, endpoint.MCPTargetsOutput]{
		{
			Name: "no_filter",
			Args: endpoint.MCPTargetsInput{},
			Expect: endpoint.MCPTargetsOutput{
				Targets: []string{
					"dummy:#no-record-yet",
					"http://a.example.com",
					"http://b.example.com",
					"http://c.example.com",
				},
			},
		},
		{
			Name: "single_keyword_1",
			Args: endpoint.MCPTargetsInput{
				Keywords: []string{"a."},
			},
			Expect: endpoint.MCPTargetsOutput{
				Targets: []string{
					"http://a.example.com",
				},
			},
		},
		{
			Name: "single_keyword_2",
			Args: endpoint.MCPTargetsInput{
				Keywords: []string{"example"},
			},
			Expect: endpoint.MCPTargetsOutput{
				Targets: []string{
					"http://a.example.com",
					"http://b.example.com",
					"http://c.example.com",
				},
			},
		},
		{
			Name: "multiple_keywords",
			Args: endpoint.MCPTargetsInput{
				Keywords: []string{"b", "c"},
			},
			Expect: endpoint.MCPTargetsOutput{
				Targets: []string{
					"http://b.example.com",
				},
			},
		},
		{
			Name: "no_match",
			Args: endpoint.MCPTargetsInput{
				Keywords: []string{"nonexistent"},
			},
			Expect: endpoint.MCPTargetsOutput{
				Targets: []string{},
			},
		},
	}

	RunMCPTest(t, "list_targets", tests)
}

func TestMCPHandler_QueryStatus(t *testing.T) {
	tests := []MCPTest[endpoint.MCPStatusInput, endpoint.MCPOutput]{
		{
			Name: "without_query",
			Args: endpoint.MCPStatusInput{},
			Expect: endpoint.MCPOutput{
				Result: map[string]any{
					"current_incidents": []any{
						map[string]any{
							"target":         "http://c.example.com",
							"status":         "UNKNOWN",
							"message":        "this is unknown",
							"starts_at":      "2021-01-02T15:04:09Z",
							"starts_at_unix": 1609599849.0,
							"ends_at":        nil,
							"ends_at_unix":   nil,
						},
					},
					"incident_history": []any{
						map[string]any{
							"target":         "http://b.example.com",
							"status":         "FAILURE",
							"message":        "this is failure",
							"starts_at":      "2021-01-02T15:04:05Z",
							"starts_at_unix": 1609599845.0,
							"ends_at":        "2021-01-02T15:04:06Z",
							"ends_at_unix":   1609599846.0,
						},
					},
					"probe_history": map[string]any{
						"dummy:#no-record-yet": map[string]any{
							"records": []any{},
							"status":  "UNKNOWN",
							"updated": nil,
						},
						"http://a.example.com": map[string]any{
							"status":  "HEALTHY",
							"updated": "2021-01-02T15:04:07Z",
							"records": []any{
								map[string]any{
									"latency":    "123.456ms",
									"latency_ms": 123.456,
									"message":    "hello world",
									"status":     "HEALTHY",
									"target":     "http://a.example.com",
									"time":       "2021-01-02T15:04:05Z",
									"time_unix":  1609599845.0,
								},
								map[string]any{
									"latency":    "234.567ms",
									"latency_ms": 234.567,
									"message":    "hello world!",
									"status":     "HEALTHY",
									"target":     "http://a.example.com",
									"time":       "2021-01-02T15:04:06Z",
									"time_unix":  1609599846.0,
								},
								map[string]any{
									"latency":    "345.678ms",
									"latency_ms": 345.678,
									"message":    "hello world!!",
									"status":     "HEALTHY",
									"target":     "http://a.example.com",
									"time":       "2021-01-02T15:04:07Z",
									"time_unix":  1609599847.0,
								},
							},
						},
						"http://b.example.com": map[string]any{
							"records": []any{
								map[string]any{
									"latency":    "12.345ms",
									"latency_ms": 12.345,
									"message":    "this is failure",
									"status":     "FAILURE",
									"target":     "http://b.example.com",
									"time":       "2021-01-02T15:04:05Z",
									"time_unix":  1609599845.0,
								},
								map[string]any{
									"extra":      1.234,
									"latency":    "54.321ms",
									"latency_ms": 54.321,
									"message":    "this is healthy",
									"status":     "HEALTHY",
									"target":     "http://b.example.com",
									"time":       "2021-01-02T15:04:06Z",
									"time_unix":  1609599846.0,
								},
							},
							"status":  "HEALTHY",
							"updated": "2021-01-02T15:04:06Z",
						},
						"http://c.example.com": map[string]any{
							"records": []any{
								map[string]any{
									"hello":      "world",
									"latency":    "1.234ms",
									"latency_ms": 1.234,
									"message":    "this is aborted",
									"status":     "ABORTED",
									"target":     "http://c.example.com",
									"time":       "2021-01-02T15:04:08Z",
									"time_unix":  1609599848.0,
								},
								map[string]any{
									"extra":      []any{1.0, 2.0, 3.0},
									"hoge":       "fuga",
									"latency":    "2.345ms",
									"latency_ms": 2.345,
									"message":    "this is unknown",
									"status":     "UNKNOWN",
									"target":     "http://c.example.com",
									"time":       "2021-01-02T15:04:09Z",
									"time_unix":  1609599849.0,
								},
							},
							"status":  "UNKNOWN",
							"updated": "2021-01-02T15:04:09Z",
						},
					},
				},
			},
		},
		{
			Name: "with_single_result_query",
			Args: endpoint.MCPStatusInput{
				Query: `.probe_history["http://a.example.com"].records[0].message`,
			},
			Expect: endpoint.MCPOutput{
				Result: "hello world",
			},
		},
		{
			Name: "with_multiple_result_query",
			Args: endpoint.MCPStatusInput{
				Query: `.probe_history["http://a.example.com"].records[] | select(.status == "HEALTHY") | .message`,
			},
			Expect: endpoint.MCPOutput{
				Result: []any{
					"hello world",
					"hello world!",
					"hello world!!",
				},
			},
		},
		{
			Name: "with_no_result_query",
			Args: endpoint.MCPStatusInput{
				Query: `.probe_history[] | select(.status == "nonexistent")`,
			},
			Expect: endpoint.MCPOutput{
				Result: nil,
			},
		},
		{
			Name: "invalid_query",
			Args: endpoint.MCPStatusInput{
				Query: `(.incident_history`,
			},
			Expect: endpoint.MCPOutput{
				Error: `failed to parse query: unexpected EOF`,
			},
		},
		{
			Name: "unknown_function",
			Args: endpoint.MCPStatusInput{
				Query: `.probe_history | unknown_function`,
			},
			Expect: endpoint.MCPOutput{
				Error: `failed to parse query: function not defined: unknown_function/0`,
			},
		},
	}

	RunMCPTest(t, "query_status", tests)
}

func TestMCPHandler_QueryLogs(t *testing.T) {
	tests := []MCPTest[endpoint.MCPLogsInput, endpoint.MCPOutput]{
		{
			Name: "without_params",
			Args: endpoint.MCPLogsInput{},
			Expect: endpoint.MCPOutput{
				Error: "since and until parameters are required",
			},
		},
		{
			Name: "without_since",
			Args: endpoint.MCPLogsInput{
				Until: "2021-01-02T15:04:10Z",
			},
			Expect: endpoint.MCPOutput{
				Error: "since and until parameters are required",
			},
		},
		{
			Name: "without_until",
			Args: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:00Z",
			},
			Expect: endpoint.MCPOutput{
				Error: "since and until parameters are required",
			},
		},
		{
			Name: "without_query",
			Args: endpoint.MCPLogsInput{
				Since: "2000-01-01T00:00:00Z",
				Until: "2100-01-01T00:00:00Z",
			},
			Expect: endpoint.MCPOutput{
				Result: []any{
					map[string]any{
						"latency":    "123.456ms",
						"latency_ms": 123.456,
						"message":    "hello world",
						"status":     "HEALTHY",
						"target":     "http://a.example.com",
						"time":       "2021-01-02T15:04:05Z",
						"time_unix":  1609599845.0,
					},
					map[string]any{
						"latency":    "12.345ms",
						"latency_ms": 12.345,
						"message":    "this is failure",
						"status":     "FAILURE",
						"target":     "http://b.example.com",
						"time":       "2021-01-02T15:04:05Z",
						"time_unix":  1609599845.0,
					},
					map[string]any{
						"latency":    "234.567ms",
						"latency_ms": 234.567,
						"message":    "hello world!",
						"status":     "HEALTHY",
						"target":     "http://a.example.com",
						"time":       "2021-01-02T15:04:06Z",
						"time_unix":  1609599846.0,
					},
					map[string]any{
						"extra":      1.234,
						"latency":    "54.321ms",
						"latency_ms": 54.321,
						"message":    "this is healthy",
						"status":     "HEALTHY",
						"target":     "http://b.example.com",
						"time":       "2021-01-02T15:04:06Z",
						"time_unix":  1609599846.0,
					},
					map[string]any{
						"latency":    "345.678ms",
						"latency_ms": 345.678,
						"message":    "hello world!!",
						"status":     "HEALTHY",
						"target":     "http://a.example.com",
						"time":       "2021-01-02T15:04:07Z",
						"time_unix":  1609599847.0,
					},
					map[string]any{
						"hello":      "world",
						"latency":    "1.234ms",
						"latency_ms": 1.234,
						"message":    "this is aborted",
						"status":     "ABORTED",
						"target":     "http://c.example.com",
						"time":       "2021-01-02T15:04:08Z",
						"time_unix":  1609599848.0,
					},
					map[string]any{
						"extra":      []any{1.0, 2.0, 3.0},
						"hoge":       "fuga",
						"latency":    "2.345ms",
						"latency_ms": 2.345,
						"message":    "this is unknown",
						"status":     "UNKNOWN",
						"target":     "http://c.example.com",
						"time":       "2021-01-02T15:04:09Z",
						"time_unix":  1609599849.0,
					},
				},
			},
		},
		{
			Name: "with_single_object_result_query",
			Args: endpoint.MCPLogsInput{
				Since: "2000-01-01T00:00:00Z",
				Until: "2100-01-01T00:00:00Z",
				Query: `.[0]`,
			},
			Expect: endpoint.MCPOutput{
				Result: map[string]any{
					"latency":    "123.456ms",
					"latency_ms": 123.456,
					"message":    "hello world",
					"status":     "HEALTHY",
					"target":     "http://a.example.com",
					"time":       "2021-01-02T15:04:05Z",
					"time_unix":  1609599845.0,
				},
			},
		},
		{
			Name: "with_single_value_result_query",
			Args: endpoint.MCPLogsInput{
				Since: "2000-01-01T00:00:00Z",
				Until: "2100-01-01T00:00:00Z",
				Query: `.[0].message`,
			},
			Expect: endpoint.MCPOutput{
				Result: "hello world",
			},
		},
		{
			Name: "with_multiple_result_query",
			Args: endpoint.MCPLogsInput{
				Since: "2000-01-01T00:00:00Z",
				Until: "2100-01-01T00:00:00Z",
				Query: `group_by(.target) | map({target: .[0].target, count: length})`,
			},
			Expect: endpoint.MCPOutput{
				Result: []any{
					map[string]any{
						"target": "http://a.example.com",
						"count":  3.0,
					},
					map[string]any{
						"target": "http://b.example.com",
						"count":  2.0,
					},
					map[string]any{
						"target": "http://c.example.com",
						"count":  2.0,
					},
				},
			},
		},
		{
			Name: "with_no_result_query",
			Args: endpoint.MCPLogsInput{
				Since: "2000-01-01T00:00:00Z",
				Until: "2100-01-01T00:00:00Z",
				Query: `.[] | select(.target == "dummy:nonexistent")`,
			},
			Expect: endpoint.MCPOutput{
				Result: nil,
			},
		},
		{
			Name: "invalid_since",
			Args: endpoint.MCPLogsInput{
				Since: "invalid-time-format",
				Until: "2021-01-02T15:04:10Z",
			},
			Expect: endpoint.MCPOutput{
				Error: `since time must be in RFC3339 format but got "invalid-time-format"`,
			},
		},
		{
			Name: "invalid_until",
			Args: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:00Z",
				Until: "invalid-time-format",
			},
			Expect: endpoint.MCPOutput{
				Error: `until time must be in RFC3339 format but got "invalid-time-format"`,
			},
		},
		{
			Name: "invalid_query",
			Args: endpoint.MCPLogsInput{
				Since: "2000-01-01T00:00:00Z",
				Until: "2100-01-01T00:00:00Z",
				Query: `.[`,
			},
			Expect: endpoint.MCPOutput{
				Error: `failed to parse query: unexpected EOF`,
			},
		},
		{
			Name: "unknown_function",
			Args: endpoint.MCPLogsInput{
				Since: "2000-01-01T00:00:00Z",
				Until: "2100-01-01T00:00:00Z",
				Query: `unknown_function(123)`,
			},
			Expect: endpoint.MCPOutput{
				Error: `failed to parse query: function not defined: unknown_function/1`,
			},
		},
	}

	RunMCPTest(t, "query_logs", tests)
}
