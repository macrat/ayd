package endpoint_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/testutil"
)

func TestFetchLogsByJq(t *testing.T) {
	tests := []struct {
		Name        string
		Input       endpoint.MCPLogsInput
		ExpectError string
		WantResult  any
	}{
		{
			Name: "empty_query_defaults_to_dot",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: "",
			},
			WantResult: []any{
				map[string]any{"time": "2021-01-02T15:04:05Z", "time_unix": int(1609599845), "status": "HEALTHY", "latency": "123.456ms", "latency_ms": 123.456, "target": "http://a.example.com", "message": "hello world"},
				map[string]any{"time": "2021-01-02T15:04:05Z", "time_unix": int(1609599845), "status": "FAILURE", "latency": "12.345ms", "latency_ms": 12.345, "target": "http://b.example.com", "message": "this is failure"},
				map[string]any{"time": "2021-01-02T15:04:06Z", "time_unix": int(1609599846), "status": "HEALTHY", "latency": "234.567ms", "latency_ms": 234.567, "target": "http://a.example.com", "message": "hello world!"},
				map[string]any{"time": "2021-01-02T15:04:06Z", "time_unix": int(1609599846), "status": "HEALTHY", "latency": "54.321ms", "latency_ms": 54.321, "target": "http://b.example.com", "message": "this is healthy", "extra": 1.234},
				map[string]any{"time": "2021-01-02T15:04:07Z", "time_unix": int(1609599847), "status": "HEALTHY", "latency": "345.678ms", "latency_ms": 345.678, "target": "http://a.example.com", "message": "hello world!!"},
				map[string]any{"time": "2021-01-02T15:04:08Z", "time_unix": int(1609599848), "status": "ABORTED", "latency": "1.234ms", "latency_ms": 1.234, "target": "http://c.example.com", "message": "this is aborted", "hello": "world"},
				map[string]any{"time": "2021-01-02T15:04:09Z", "time_unix": int(1609599849), "status": "UNKNOWN", "latency": "2.345ms", "latency_ms": 2.345, "target": "http://c.example.com", "message": "this is unknown", "hoge": "fuga", "extra": []any{float64(1), float64(2), float64(3)}},
			},
		},
		{
			Name: "dot_query_returns_all_records",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: ".",
			},
			WantResult: []any{
				map[string]any{"time": "2021-01-02T15:04:05Z", "time_unix": int(1609599845), "status": "HEALTHY", "latency": "123.456ms", "latency_ms": 123.456, "target": "http://a.example.com", "message": "hello world"},
				map[string]any{"time": "2021-01-02T15:04:05Z", "time_unix": int(1609599845), "status": "FAILURE", "latency": "12.345ms", "latency_ms": 12.345, "target": "http://b.example.com", "message": "this is failure"},
				map[string]any{"time": "2021-01-02T15:04:06Z", "time_unix": int(1609599846), "status": "HEALTHY", "latency": "234.567ms", "latency_ms": 234.567, "target": "http://a.example.com", "message": "hello world!"},
				map[string]any{"time": "2021-01-02T15:04:06Z", "time_unix": int(1609599846), "status": "HEALTHY", "latency": "54.321ms", "latency_ms": 54.321, "target": "http://b.example.com", "message": "this is healthy", "extra": 1.234},
				map[string]any{"time": "2021-01-02T15:04:07Z", "time_unix": int(1609599847), "status": "HEALTHY", "latency": "345.678ms", "latency_ms": 345.678, "target": "http://a.example.com", "message": "hello world!!"},
				map[string]any{"time": "2021-01-02T15:04:08Z", "time_unix": int(1609599848), "status": "ABORTED", "latency": "1.234ms", "latency_ms": 1.234, "target": "http://c.example.com", "message": "this is aborted", "hello": "world"},
				map[string]any{"time": "2021-01-02T15:04:09Z", "time_unix": int(1609599849), "status": "UNKNOWN", "latency": "2.345ms", "latency_ms": 2.345, "target": "http://c.example.com", "message": "this is unknown", "hoge": "fuga", "extra": []any{float64(1), float64(2), float64(3)}},
			},
		},
		{
			Name: "length_query",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: "length",
			},
			// length query returns a single number (not an array)
			WantResult: 7,
		},
		{
			Name: "filter_by_status_healthy",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: `.[] | select(.status == "HEALTHY")`,
			},
			WantResult: []any{
				map[string]any{"time": "2021-01-02T15:04:05Z", "time_unix": int(1609599845), "status": "HEALTHY", "latency": "123.456ms", "latency_ms": 123.456, "target": "http://a.example.com", "message": "hello world"},
				map[string]any{"time": "2021-01-02T15:04:06Z", "time_unix": int(1609599846), "status": "HEALTHY", "latency": "234.567ms", "latency_ms": 234.567, "target": "http://a.example.com", "message": "hello world!"},
				map[string]any{"time": "2021-01-02T15:04:06Z", "time_unix": int(1609599846), "status": "HEALTHY", "latency": "54.321ms", "latency_ms": 54.321, "target": "http://b.example.com", "message": "this is healthy", "extra": 1.234},
				map[string]any{"time": "2021-01-02T15:04:07Z", "time_unix": int(1609599847), "status": "HEALTHY", "latency": "345.678ms", "latency_ms": 345.678, "target": "http://a.example.com", "message": "hello world!!"},
			},
		},
		{
			Name: "filter_by_target",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: `.[] | select(.target == "http://c.example.com")`,
			},
			WantResult: []any{
				map[string]any{"time": "2021-01-02T15:04:08Z", "time_unix": int(1609599848), "status": "ABORTED", "latency": "1.234ms", "latency_ms": 1.234, "target": "http://c.example.com", "message": "this is aborted", "hello": "world"},
				map[string]any{"time": "2021-01-02T15:04:09Z", "time_unix": int(1609599849), "status": "UNKNOWN", "latency": "2.345ms", "latency_ms": 2.345, "target": "http://c.example.com", "message": "this is unknown", "hoge": "fuga", "extra": []any{float64(1), float64(2), float64(3)}},
			},
		},
		{
			Name: "map_to_subset_fields",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: `.[] | {target, status, latency_ms}`,
			},
			WantResult: []any{
				map[string]any{"target": "http://a.example.com", "status": "HEALTHY", "latency_ms": 123.456},
				map[string]any{"target": "http://b.example.com", "status": "FAILURE", "latency_ms": 12.345},
				map[string]any{"target": "http://a.example.com", "status": "HEALTHY", "latency_ms": 234.567},
				map[string]any{"target": "http://b.example.com", "status": "HEALTHY", "latency_ms": 54.321},
				map[string]any{"target": "http://a.example.com", "status": "HEALTHY", "latency_ms": 345.678},
				map[string]any{"target": "http://c.example.com", "status": "ABORTED", "latency_ms": 1.234},
				map[string]any{"target": "http://c.example.com", "status": "UNKNOWN", "latency_ms": 2.345},
			},
		},
		{
			Name: "empty_result_when_no_match",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: `.[] | select(.status == "NONEXISTENT")`,
			},
			// No matches returns nil (empty slice is never appended to)
			WantResult: nil,
		},
		{
			Name: "empty_result_when_no_logs_in_range",
			Input: endpoint.MCPLogsInput{
				Since: "2020-01-01T00:00:00Z",
				Until: "2020-01-02T00:00:00Z",
				Query: ".",
			},
			// Empty time range returns empty array (single result with 0 elements)
			WantResult: []any{},
		},
		{
			Name: "time_range_filtering",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:06Z",
				Until: "2021-01-02T15:04:08Z",
				Query: ".",
			},
			// Should get records at 15:04:06 and 15:04:07 (2 + 1 = 3 records)
			WantResult: []any{
				map[string]any{
					"time":       "2021-01-02T15:04:06Z",
					"time_unix":  int(1609599846),
					"status":     "HEALTHY",
					"latency":    "234.567ms",
					"latency_ms": 234.567,
					"target":     "http://a.example.com",
					"message":    "hello world!",
				},
				map[string]any{
					"time":       "2021-01-02T15:04:06Z",
					"time_unix":  int(1609599846),
					"status":     "HEALTHY",
					"latency":    "54.321ms",
					"latency_ms": 54.321,
					"target":     "http://b.example.com",
					"message":    "this is healthy",
					"extra":      1.234,
				},
				map[string]any{
					"time":       "2021-01-02T15:04:07Z",
					"time_unix":  int(1609599847),
					"status":     "HEALTHY",
					"latency":    "345.678ms",
					"latency_ms": 345.678,
					"target":     "http://a.example.com",
					"message":    "hello world!!",
				},
			},
		},
		{
			Name: "invalid_since_time",
			Input: endpoint.MCPLogsInput{
				Since: "invalid-time",
				Until: "2021-01-02T15:04:10Z",
				Query: ".",
			},
			ExpectError: "invalid since time",
		},
		{
			Name: "invalid_until_time",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "invalid-time",
				Query: ".",
			},
			ExpectError: "invalid until time",
		},
		{
			Name: "invalid_jq_query",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: ".[ invalid syntax",
			},
			ExpectError: "failed to parse query",
		},
		{
			Name: "jq_type_error",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: ".nonexistent_field.nested",
			},
			// This causes a jq runtime error when trying to access nested field on array
			ExpectError: "expected an object but got",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			s := testutil.NewStoreWithLog(t)
			ctx := context.Background()

			output := endpoint.FetchLogsByJq(ctx, s, tt.Input)

			if tt.ExpectError != "" {
				if output.Error == "" {
					t.Errorf("expected error containing %q, got no error", tt.ExpectError)
				} else if !containsString(output.Error, tt.ExpectError) {
					t.Errorf("expected error containing %q, got %q", tt.ExpectError, output.Error)
				}
				// On error, Result may be nil
			} else {
				if output.Error != "" {
					t.Errorf("unexpected error: %s", output.Error)
				}
				if output.Result == nil {
					t.Fatal("Result should not be nil")
				}
				if tt.WantResult != nil {
					if diff := cmp.Diff(tt.WantResult, output.Result); diff != "" {
						t.Errorf("Result mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

func TestFetchStatusByJq(t *testing.T) {
	tests := []struct {
		Name        string
		Input       endpoint.MCPStatusInput
		ExpectError string
		WantResult  any
	}{
		{
			Name: "empty_query_defaults_to_dot",
			Input: endpoint.MCPStatusInput{
				Query: "",
			},
			WantResult: map[string]any{
				"current_incidents": []any{},
				"incident_history":  []any{},
				"probe_history": map[string]any{
					"dummy:#no-record-yet": map[string]any{
						"records": []any{},
						"status":  "UNKNOWN",
						"updated": "0001-01-01T00:00:00Z",
					},
				},
			},
		},
		{
			Name: "dot_query_returns_full_status",
			Input: endpoint.MCPStatusInput{
				Query: ".",
			},
			WantResult: map[string]any{
				"current_incidents": []any{},
				"incident_history":  []any{},
				"probe_history": map[string]any{
					"dummy:#no-record-yet": map[string]any{
						"records": []any{},
						"status":  "UNKNOWN",
						"updated": "0001-01-01T00:00:00Z",
					},
				},
			},
		},
		{
			Name: "get_probe_history_keys",
			Input: endpoint.MCPStatusInput{
				Query: ".probe_history | keys",
			},
			WantResult: []any{"dummy:#no-record-yet"},
		},
		{
			Name: "get_current_incidents",
			Input: endpoint.MCPStatusInput{
				Query: ".current_incidents",
			},
			WantResult: []any{},
		},
		{
			Name: "map_probe_history",
			Input: endpoint.MCPStatusInput{
				Query: `.probe_history | to_entries | map({target: .key, status: .value.status})`,
			},
			WantResult: []any{
				map[string]any{"target": "dummy:#no-record-yet", "status": "UNKNOWN"},
			},
		},
		{
			Name: "filter_by_status",
			Input: endpoint.MCPStatusInput{
				Query: `.probe_history | to_entries | map(select(.value.status == "HEALTHY")) | map(.key)`,
			},
			WantResult: []any{},
		},
		{
			Name: "invalid_jq_query",
			Input: endpoint.MCPStatusInput{
				Query: ".[ invalid syntax",
			},
			ExpectError: "failed to parse query",
		},
		{
			Name: "jq_type_error",
			Input: endpoint.MCPStatusInput{
				Query: ".probe_history | keys | .[0] | .nonexistent",
			},
			// This causes a runtime error: trying to access object field on string
			ExpectError: "expected an object but got",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			s := testutil.NewStoreWithLog(t)
			ctx := context.Background()

			output := endpoint.FetchStatusByJq(ctx, s, tt.Input)

			if tt.ExpectError != "" {
				if output.Error == "" {
					t.Errorf("expected error containing %q, got no error", tt.ExpectError)
				} else if !containsString(output.Error, tt.ExpectError) {
					t.Errorf("expected error containing %q, got %q", tt.ExpectError, output.Error)
				}
				// On error, Result may be nil
			} else {
				if output.Error != "" {
					t.Errorf("unexpected error: %s", output.Error)
				}
				if output.Result == nil {
					t.Fatal("Result should not be nil")
				}
				if tt.WantResult != nil {
					if diff := cmp.Diff(tt.WantResult, output.Result); diff != "" {
						t.Errorf("Result mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

func TestRecordToMapFields(t *testing.T) {
	// recordToMap is an unexported function, but we can test it indirectly
	// through FetchLogsByJq. This test verifies the structure of converted records.
	s := testutil.NewStoreWithLog(t)
	ctx := context.Background()

	// Test 1: Verify basic record structure with .[0]
	output := endpoint.FetchLogsByJq(ctx, s, endpoint.MCPLogsInput{
		Since: "2021-01-02T15:04:05Z",
		Until: "2021-01-02T15:04:06Z",
		Query: ".[0]",
	})

	if output.Error != "" {
		t.Fatalf("unexpected error: %s", output.Error)
	}

	wantRecord := map[string]any{
		"time":       "2021-01-02T15:04:05Z",
		"time_unix":  int(1609599845),
		"status":     "HEALTHY",
		"latency":    "123.456ms",
		"latency_ms": 123.456,
		"target":     "http://a.example.com",
		"message":    "hello world",
	}

	if diff := cmp.Diff(wantRecord, output.Result); diff != "" {
		t.Errorf("Record structure mismatch (-want +got):\n%s", diff)
	}

	// Test 2: Verify that extra fields are copied
	output2 := endpoint.FetchLogsByJq(ctx, s, endpoint.MCPLogsInput{
		Since: "2021-01-02T15:04:06Z",
		Until: "2021-01-02T15:04:07Z",
		Query: `.[] | select(.target == "http://b.example.com")`,
	})

	if output2.Error != "" {
		t.Fatalf("unexpected error: %s", output2.Error)
	}

	wantRecordWithExtra := map[string]any{
		"time":       "2021-01-02T15:04:06Z",
		"time_unix":  int(1609599846),
		"status":     "HEALTHY",
		"latency":    "54.321ms",
		"latency_ms": 54.321,
		"target":     "http://b.example.com",
		"message":    "this is healthy",
		"extra":      1.234,
	}

	if diff := cmp.Diff(wantRecordWithExtra, output2.Result); diff != "" {
		t.Errorf("Record with extra fields mismatch (-want +got):\n%s", diff)
	}
}

func TestFetchTargets(t *testing.T) {
	tests := []struct {
		Name            string
		Input           endpoint.MCPTargetsInput
		ExpectedCount   int
		ExpectedTargets []string // Expected targets to be present (can be subset)
		MustBeExact     bool     // If true, ExpectedTargets must match exactly
	}{
		{
			Name:          "no_keywords_returns_all_targets",
			Input:         endpoint.MCPTargetsInput{Keywords: []string{}},
			ExpectedCount: 4, // a, b, c, and dummy
		},
		{
			Name:          "nil_keywords_returns_all_targets",
			Input:         endpoint.MCPTargetsInput{Keywords: nil},
			ExpectedCount: 4,
		},
		{
			Name:  "filter_by_http",
			Input: endpoint.MCPTargetsInput{Keywords: []string{"http"}},
			ExpectedTargets: []string{
				"http://a.example.com",
				"http://b.example.com",
				"http://c.example.com",
			},
			MustBeExact: true,
		},
		{
			Name:  "filter_by_example",
			Input: endpoint.MCPTargetsInput{Keywords: []string{"example"}},
			ExpectedTargets: []string{
				"http://a.example.com",
				"http://b.example.com",
				"http://c.example.com",
			},
			MustBeExact: true,
		},
		{
			Name:  "filter_by_multiple_keywords_http_and_specific_a",
			Input: endpoint.MCPTargetsInput{Keywords: []string{"http", "://a."}},
			ExpectedTargets: []string{
				"http://a.example.com",
			},
			MustBeExact: true,
		},
		{
			Name:  "filter_by_multiple_keywords_specific_b",
			Input: endpoint.MCPTargetsInput{Keywords: []string{".com", "://b"}},
			ExpectedTargets: []string{
				"http://b.example.com",
			},
			MustBeExact: true,
		},
		{
			Name:            "filter_by_non_matching_keyword",
			Input:           endpoint.MCPTargetsInput{Keywords: []string{"https"}},
			ExpectedTargets: []string{},
			MustBeExact:     true,
		},
		{
			Name:  "filter_by_dummy",
			Input: endpoint.MCPTargetsInput{Keywords: []string{"dummy"}},
			ExpectedTargets: []string{
				"dummy:#no-record-yet",
			},
			MustBeExact: true,
		},
		{
			Name:            "filter_by_conflicting_keywords",
			Input:           endpoint.MCPTargetsInput{Keywords: []string{"http", "dummy"}},
			ExpectedTargets: []string{},
			MustBeExact:     true,
		},
		{
			Name:  "filter_by_b_example",
			Input: endpoint.MCPTargetsInput{Keywords: []string{"b", "example"}},
			ExpectedTargets: []string{
				"http://b.example.com",
			},
			MustBeExact: true,
		},
		{
			Name:  "filter_by_partial_match",
			Input: endpoint.MCPTargetsInput{Keywords: []string{"exam"}},
			ExpectedTargets: []string{
				"http://a.example.com",
				"http://b.example.com",
				"http://c.example.com",
			},
			MustBeExact: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			s := testutil.NewStoreWithLog(t)
			ctx := context.Background()

			output := endpoint.FetchTargets(ctx, s, tt.Input)

			if output.Targets == nil {
				t.Fatal("Targets should not be nil")
			}

			if tt.ExpectedCount > 0 {
				if len(output.Targets) != tt.ExpectedCount {
					t.Errorf("expected %d targets, got %d: %v", tt.ExpectedCount, len(output.Targets), output.Targets)
				}
			}

			if len(tt.ExpectedTargets) > 0 {
				if tt.MustBeExact {
					if len(output.Targets) != len(tt.ExpectedTargets) {
						t.Errorf("expected exactly %d targets, got %d: %v", len(tt.ExpectedTargets), len(output.Targets), output.Targets)
					}
					// Check all expected targets are present
					for _, expected := range tt.ExpectedTargets {
						found := false
						for _, actual := range output.Targets {
							if actual == expected {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected target %q not found in results: %v", expected, output.Targets)
						}
					}
					// Check no unexpected targets are present
					for _, actual := range output.Targets {
						found := false
						for _, expected := range tt.ExpectedTargets {
							if actual == expected {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("unexpected target %q found in results", actual)
						}
					}
				} else {
					// Just check expected targets are present (subset check)
					for _, expected := range tt.ExpectedTargets {
						found := false
						for _, actual := range output.Targets {
							if actual == expected {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected target %q not found in results: %v", expected, output.Targets)
						}
					}
				}
			}
		})
	}
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
