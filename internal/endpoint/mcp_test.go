package endpoint_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/scheme"
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
			Query:  `.nonexistent`,
			Expect: nil,
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
		{
			Query:  `halt`,
			Expect: []any(nil),
		},
		{
			Query: `halt_error`,
			Expect: map[string]any{
				"status":    "halt_error",
				"exit_code": 5,
				"value":     input,
			},
		},
		{
			Query: `123 | halt_error(4)`,
			Expect: map[string]any{
				"status":    "halt_error",
				"exit_code": 4,
				"value":     123,
			},
		},
		{
			Query: `error("hello")`,
			Error: `error: hello`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Query, func(t *testing.T) {
			s := testutil.NewStore(t)

			jq, err := endpoint.ParseJQ(tt.Query)
			if err != nil {
				t.Fatalf("failed to parse JQ query: %v", err)
			}

			output := jq.Run(context.Background(), s, "mcp/test", input)

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

func TestMCP_connection(t *testing.T) {
	tests := []struct {
		Name             string
		WithInstanceName bool
	}{
		{"without_instance_name", false},
		{"with_instance_name", true},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var opts []testutil.StoreOption
			if tt.WithInstanceName {
				opts = append(opts, testutil.WithInstanceName("test-instance"))
			}

			srv := testutil.StartTestServer(t, opts...)
			t.Cleanup(func() {
				srv.Close()
			})

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

			initResult := sess.InitializeResult()
			if initResult.ServerInfo.Name != "ayd" {
				t.Errorf("unexpected server name: %q", initResult.ServerInfo.Name)
			}
			if tt.WithInstanceName {
				if initResult.ServerInfo.Title != "Ayd (test-instance)" {
					t.Errorf("unexpected server title: %q", initResult.ServerInfo.Title)
				}
				if !strings.Contains(initResult.Instructions, "test-instance") {
					t.Errorf("instructions does not contain instance name: %q", initResult.Instructions)
				}
			} else {
				if initResult.ServerInfo.Title != "Ayd" {
					t.Errorf("unexpected server title: %q", initResult.ServerInfo.Title)
				}
				if strings.Contains(initResult.Instructions, "instance") {
					t.Errorf("instructions contains instance name: %q", initResult.Instructions)
				}
			}

			if err := sess.Ping(t.Context(), nil); err != nil {
				t.Fatalf("failed to ping MCP server: %v", err)
			}
		})
	}
}

func NewTestMCPServer(tb testing.TB, s endpoint.Store) *mcp.ClientSession {
	srvPort, cliPort := mcp.NewInMemoryTransports()

	srv := endpoint.MCPServer(s)
	srv.Connect(tb.Context(), srvPort, nil)

	cli := mcp.NewClient(&mcp.Implementation{
		Name:    "benchmark-client",
		Version: "none",
	}, nil)
	sess, err := cli.Connect(tb.Context(), cliPort, nil)
	if err != nil {
		tb.Fatalf("failed to connect to MCP server: %v", err)
	}

	return sess
}

func BenchmarkMCPHandler_QueryStatus(b *testing.B) {
	sess := NewTestMCPServer(b, testutil.NewStore(b))

	for b.Loop() {
		_, err := sess.CallTool(b.Context(), &mcp.CallToolParams{
			Name: "query_status",
			Arguments: endpoint.MCPStatusInput{
				Query: `.probe_history | to_entries[] | {target: .key, status: .value.status}`,
			},
		})
		if err != nil {
			b.Fatalf("failed to call tool query_status: %v", err)
		}
	}
}

func BenchmarkMCPHandler_QueryLogs_smallLogs(b *testing.B) {
	sess := NewTestMCPServer(b, testutil.NewStore(b))

	for b.Loop() {
		_, err := sess.CallTool(b.Context(), &mcp.CallToolParams{
			Name: "query_logs",
			Arguments: endpoint.MCPLogsInput{
				Since: "2000-01-01T00:00:00Z",
				Until: "2100-01-01T00:00:00Z",
				Query: `group_by(.target)[] | {target: .[0].target, count: length}`,
			},
		})
		if err != nil {
			b.Fatalf("failed to call tool query_logs: %v", err)
		}
	}
}

func BenchmarkMCPHandler_QueryLogs_largeLogs(b *testing.B) {
	s := testutil.NewStore(b)

	var probers []scheme.Prober
	for i := range 10 {
		probers = append(probers, testutil.NewProber(b, fmt.Sprintf("dummy://random?latency=0ms#%d", i)))
	}

	for range 100_000 {
		for _, p := range probers {
			p.Probe(context.Background(), s)
		}
	}

	sess := NewTestMCPServer(b, s)

	for b.Loop() {
		_, err := sess.CallTool(b.Context(), &mcp.CallToolParams{
			Name: "query_logs",
			Arguments: endpoint.MCPLogsInput{
				Since: "2000-01-01T00:00:00Z",
				Until: "2100-01-01T00:00:00Z",
				Query: `[.[] | select(.status == "HEALTHY")] | group_by(.target)[] | {target: .[0].target, count: length}`,
			},
		})
		if err != nil {
			b.Fatalf("failed to call tool query_logs: %v", err)
		}
	}
}
