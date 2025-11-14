package endpoint_test

import (
	"context"
	"testing"

	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/testutil"
)

func TestFetchLogsByJq(t *testing.T) {
	tests := []struct {
		Name         string
		Input        endpoint.MCPLogsInput
		ExpectError  string
		CheckResults func(t *testing.T, results []any)
	}{
		{
			Name: "empty_query_defaults_to_dot",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: "",
			},
			CheckResults: func(t *testing.T, results []any) {
				if len(results) != 7 {
					t.Errorf("expected 7 results, got %d", len(results))
				}
			},
		},
		{
			Name: "dot_query_returns_all_records",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: ".",
			},
			CheckResults: func(t *testing.T, results []any) {
				if len(results) != 7 {
					t.Errorf("expected 7 results, got %d", len(results))
				}
				// Check first record structure
				if len(results) > 0 {
					rec, ok := results[0].(map[string]any)
					if !ok {
						t.Fatal("expected map[string]any")
					}
					if rec["target"] != "http://a.example.com" {
						t.Errorf("unexpected target: %v", rec["target"])
					}
					if rec["status"] != "HEALTHY" {
						t.Errorf("unexpected status: %v", rec["status"])
					}
					if rec["message"] != "hello world" {
						t.Errorf("unexpected message: %v", rec["message"])
					}
				}
			},
		},
		{
			Name: "length_query",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: "length",
			},
			CheckResults: func(t *testing.T, results []any) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				// gojq can return int or float64 for numbers
				var count int
				switch v := results[0].(type) {
				case int:
					count = v
				case float64:
					count = int(v)
				default:
					t.Fatalf("expected number type, got %T", results[0])
				}
				if count != 7 {
					t.Errorf("expected count 7, got %v", count)
				}
			},
		},
		{
			Name: "filter_by_status_healthy",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: `.[] | select(.status == "HEALTHY")`,
			},
			CheckResults: func(t *testing.T, results []any) {
				if len(results) != 4 {
					t.Errorf("expected 4 HEALTHY results, got %d", len(results))
				}
				for i, r := range results {
					rec, ok := r.(map[string]any)
					if !ok {
						t.Errorf("result[%d]: expected map[string]any", i)
						continue
					}
					if rec["status"] != "HEALTHY" {
						t.Errorf("result[%d]: expected status HEALTHY, got %v", i, rec["status"])
					}
				}
			},
		},
		{
			Name: "filter_by_target",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: `.[] | select(.target == "http://c.example.com")`,
			},
			CheckResults: func(t *testing.T, results []any) {
				if len(results) != 2 {
					t.Errorf("expected 2 results for http://c.example.com, got %d", len(results))
				}
			},
		},
		{
			Name: "map_to_subset_fields",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: `.[] | {target, status, latency_ms}`,
			},
			CheckResults: func(t *testing.T, results []any) {
				if len(results) != 7 {
					t.Errorf("expected 7 results, got %d", len(results))
				}
				if len(results) > 0 {
					rec, ok := results[0].(map[string]any)
					if !ok {
						t.Fatal("expected map[string]any")
					}
					// Check only mapped fields exist
					if _, ok := rec["target"]; !ok {
						t.Error("expected 'target' field")
					}
					if _, ok := rec["status"]; !ok {
						t.Error("expected 'status' field")
					}
					if _, ok := rec["latency_ms"]; !ok {
						t.Error("expected 'latency_ms' field")
					}
					if _, ok := rec["message"]; ok {
						t.Error("unexpected 'message' field (should be filtered out)")
					}
				}
			},
		},
		{
			Name: "empty_result_when_no_match",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:05Z",
				Until: "2021-01-02T15:04:10Z",
				Query: `.[] | select(.status == "NONEXISTENT")`,
			},
			CheckResults: func(t *testing.T, results []any) {
				if len(results) != 0 {
					t.Errorf("expected 0 results, got %d", len(results))
				}
			},
		},
		{
			Name: "empty_result_when_no_logs_in_range",
			Input: endpoint.MCPLogsInput{
				Since: "2020-01-01T00:00:00Z",
				Until: "2020-01-02T00:00:00Z",
				Query: ".",
			},
			CheckResults: func(t *testing.T, results []any) {
				if len(results) != 0 {
					t.Errorf("expected 0 results for empty time range, got %d", len(results))
				}
			},
		},
		{
			Name: "time_range_filtering",
			Input: endpoint.MCPLogsInput{
				Since: "2021-01-02T15:04:06Z",
				Until: "2021-01-02T15:04:08Z",
				Query: ".",
			},
			CheckResults: func(t *testing.T, results []any) {
				// Should get records at 15:04:06 and 15:04:07 (2 + 1 = 3 records)
				if len(results) != 3 {
					t.Errorf("expected 3 results, got %d", len(results))
				}
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
				// Result should always be non-nil array even on error
				if output.Result == nil {
					t.Error("Result should be non-nil array even on error")
				}
			} else {
				if output.Error != "" {
					t.Errorf("unexpected error: %s", output.Error)
				}
				if output.Result == nil {
					t.Fatal("Result should not be nil")
				}
				if tt.CheckResults != nil {
					tt.CheckResults(t, output.Result)
				}
			}
		})
	}
}

func TestFetchStatusByJq(t *testing.T) {
	tests := []struct {
		Name         string
		Input        endpoint.MCPStatusInput
		ExpectError  string
		CheckResults func(t *testing.T, results []any)
	}{
		{
			Name: "empty_query_defaults_to_dot",
			Input: endpoint.MCPStatusInput{
				Query: "",
			},
			CheckResults: func(t *testing.T, results []any) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				obj, ok := results[0].(map[string]any)
				if !ok {
					t.Fatal("expected map[string]any")
				}
				if _, ok := obj["probe_history"]; !ok {
					t.Error("expected 'probe_history' field")
				}
				if _, ok := obj["current_incidents"]; !ok {
					t.Error("expected 'current_incidents' field")
				}
				if _, ok := obj["incident_history"]; !ok {
					t.Error("expected 'incident_history' field")
				}
				if _, ok := obj["reported_at"]; !ok {
					t.Error("expected 'reported_at' field")
				}
			},
		},
		{
			Name: "dot_query_returns_full_status",
			Input: endpoint.MCPStatusInput{
				Query: ".",
			},
			CheckResults: func(t *testing.T, results []any) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
			},
		},
		{
			Name: "get_probe_history_keys",
			Input: endpoint.MCPStatusInput{
				Query: ".probe_history | keys",
			},
			CheckResults: func(t *testing.T, results []any) {
				// keys returns array, which gets flattened by our logic
				// The flattening happens when result has exactly 1 element and it's a []any
				if len(results) < 1 {
					t.Errorf("expected at least 1 probe target key, got %d", len(results))
				}
				// Each result should be a string (the key name)
				for i, r := range results {
					if _, ok := r.(string); !ok {
						t.Errorf("result[%d]: expected string, got %T", i, r)
						break
					}
				}
			},
		},
		{
			Name: "get_current_incidents",
			Input: endpoint.MCPStatusInput{
				Query: ".current_incidents",
			},
			CheckResults: func(t *testing.T, results []any) {
				// Results should be an array (may be empty)
				if results == nil {
					t.Fatal("results should not be nil")
				}
			},
		},
		{
			Name: "map_probe_history",
			Input: endpoint.MCPStatusInput{
				Query: `.probe_history | to_entries | map({target: .key, status: .value.status})`,
			},
			CheckResults: func(t *testing.T, results []any) {
				if len(results) == 0 {
					t.Fatal("expected at least one probe")
				}
				// Check structure of first result
				probe, ok := results[0].(map[string]any)
				if !ok {
					t.Fatalf("expected map[string]any, got %T", results[0])
				}
				if _, ok := probe["target"]; !ok {
					t.Error("expected 'target' field")
				}
				if _, ok := probe["status"]; !ok {
					t.Error("expected 'status' field")
				}
			},
		},
		{
			Name: "filter_by_status",
			Input: endpoint.MCPStatusInput{
				Query: `.probe_history | to_entries | map(select(.value.status == "HEALTHY")) | map(.key)`,
			},
			CheckResults: func(t *testing.T, results []any) {
				// Should return array of target names with HEALTHY status
				if results == nil {
					t.Fatal("results should not be nil")
				}
			},
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
				// Result should always be non-nil array even on error
				if output.Result == nil {
					t.Error("Result should be non-nil array even on error")
				}
			} else {
				if output.Error != "" {
					t.Errorf("unexpected error: %s", output.Error)
				}
				if output.Result == nil {
					t.Fatal("Result should not be nil")
				}
				if tt.CheckResults != nil {
					tt.CheckResults(t, output.Result)
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

	output := endpoint.FetchLogsByJq(ctx, s, endpoint.MCPLogsInput{
		Since: "2021-01-02T15:04:05Z",
		Until: "2021-01-02T15:04:06Z",
		Query: ".[0]",
	})

	if output.Error != "" {
		t.Fatalf("unexpected error: %s", output.Error)
	}

	if len(output.Result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(output.Result))
	}

	rec, ok := output.Result[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", output.Result[0])
	}

	// Verify all expected fields are present
	expectedFields := []string{"time", "time_unix", "status", "latency", "latency_ms", "target", "message"}
	for _, field := range expectedFields {
		if _, ok := rec[field]; !ok {
			t.Errorf("missing expected field: %s", field)
		}
	}

	// Verify time format
	if timeStr, ok := rec["time"].(string); !ok {
		t.Error("time field should be string")
	} else if len(timeStr) == 0 {
		t.Error("time field should not be empty")
	}

	// Verify time_unix is a number (can be int or float64)
	switch rec["time_unix"].(type) {
	case int, int64, float64:
		// OK
	default:
		t.Errorf("time_unix field should be number, got %T", rec["time_unix"])
	}

	// Verify extra fields are copied
	output2 := endpoint.FetchLogsByJq(ctx, s, endpoint.MCPLogsInput{
		Since: "2021-01-02T15:04:06Z",
		Until: "2021-01-02T15:04:07Z",
		Query: `.[] | select(.target == "http://b.example.com")`,
	})

	if len(output2.Result) > 0 {
		rec2, ok := output2.Result[0].(map[string]any)
		if !ok {
			t.Fatal("expected map[string]any")
		}
		// This record has "extra" field in test data
		if _, ok := rec2["extra"]; !ok {
			t.Error("extra fields from Record.Extra should be copied to output")
		}
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
