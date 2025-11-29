package mcp_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/mcp"
	api "github.com/macrat/ayd/lib-ayd"
)

// mockStore implements mcp.Store for testing.
type mockStore struct {
	name            string
	probeHistory    []api.ProbeHistory
	currentIncident []*api.Incident
	incidentHistory []*api.Incident
	logs            []api.Record
	logError        error
}

func (s *mockStore) Name() string {
	return s.name
}

func (s *mockStore) ProbeHistory() []api.ProbeHistory {
	return s.probeHistory
}

func (s *mockStore) CurrentIncidents() []*api.Incident {
	return s.currentIncident
}

func (s *mockStore) IncidentHistory() []*api.Incident {
	return s.incidentHistory
}

func (s *mockStore) ReportInternalError(scope, message string) {
	// no-op for test
}

func (s *mockStore) OpenLog(since, until time.Time) (api.LogScanner, error) {
	if s.logError != nil {
		return nil, s.logError
	}
	return &mockLogScanner{logs: s.logs}, nil
}

// mockLogScanner implements api.LogScanner for testing.
type mockLogScanner struct {
	logs  []api.Record
	index int
}

func (s *mockLogScanner) Scan() bool {
	if s.index < len(s.logs) {
		s.index++
		return true
	}
	return false
}

func (s *mockLogScanner) Record() api.Record {
	return s.logs[s.index-1]
}

func (s *mockLogScanner) Close() error {
	return nil
}

func TestFetchStatusByJQ(t *testing.T) {
	target1, _ := api.ParseURL("https://example.com")
	target2, _ := api.ParseURL("https://example.org")

	store := &mockStore{
		name: "test",
		probeHistory: []api.ProbeHistory{
			{
				Target: target1,
				Status: api.StatusHealthy,
				Records: []api.Record{
					{
						Time:    time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
						Status:  api.StatusHealthy,
						Target:  target1,
						Latency: 100 * time.Millisecond,
						Message: "ok",
					},
				},
			},
			{
				Target: target2,
				Status: api.StatusFailure,
				Records: []api.Record{
					{
						Time:    time.Date(2021, 1, 2, 3, 4, 6, 0, time.UTC),
						Status:  api.StatusFailure,
						Target:  target2,
						Latency: 200 * time.Millisecond,
						Message: "connection refused",
					},
				},
			},
		},
	}

	t.Run("no_filter", func(t *testing.T) {
		output, err := mcp.FetchStatusByJQ(context.Background(), store, mcp.StatusInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		results, ok := output.Result.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", output.Result)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
	})

	t.Run("filter_healthy", func(t *testing.T) {
		output, err := mcp.FetchStatusByJQ(context.Background(), store, mcp.StatusInput{
			JQ: `.[] | select(.status == "HEALTHY")`,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, ok := output.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", output.Result)
		}
		if result["status"] != "HEALTHY" {
			t.Errorf("expected HEALTHY status, got %v", result["status"])
		}
	})

	t.Run("count_by_status", func(t *testing.T) {
		output, err := mcp.FetchStatusByJQ(context.Background(), store, mcp.StatusInput{
			JQ: `group_by(.status) | map({status: .[0].status, count: length})`,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		results, ok := output.Result.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", output.Result)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 groups, got %d", len(results))
		}
	})

	t.Run("invalid_jq", func(t *testing.T) {
		_, err := mcp.FetchStatusByJQ(context.Background(), store, mcp.StatusInput{
			JQ: "invalid{{",
		})
		if err == nil {
			t.Error("expected error for invalid jq query")
		}
	})
}

func TestFetchIncidentsByJQ(t *testing.T) {
	target, _ := api.ParseURL("https://example.com")

	store := &mockStore{
		name: "test",
		currentIncident: []*api.Incident{
			{
				Target:   target,
				Status:   api.StatusFailure,
				Message:  "ongoing error",
				StartsAt: time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
			},
		},
		incidentHistory: []*api.Incident{
			{
				Target:   target,
				Status:   api.StatusFailure,
				Message:  "resolved error",
				StartsAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndsAt:   time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
			},
		},
	}

	t.Run("ongoing_only_default", func(t *testing.T) {
		output, err := mcp.FetchIncidentsByJQ(context.Background(), store, mcp.IncidentsInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Single result may be unwrapped or wrapped in array
		var result map[string]any
		switch v := output.Result.(type) {
		case map[string]any:
			result = v
		case []any:
			if len(v) != 1 {
				t.Fatalf("expected 1 incident, got %d", len(v))
			}
			result = v[0].(map[string]any)
		default:
			t.Fatalf("unexpected type: %T", output.Result)
		}
		if result["message"] != "ongoing error" {
			t.Errorf("expected 'ongoing error', got %v", result["message"])
		}
	})

	t.Run("include_resolved", func(t *testing.T) {
		output, err := mcp.FetchIncidentsByJQ(context.Background(), store, mcp.IncidentsInput{
			IncludeResolved: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		results, ok := output.Result.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", output.Result)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 incidents, got %d", len(results))
		}
	})

	t.Run("exclude_ongoing", func(t *testing.T) {
		falseVal := false
		output, err := mcp.FetchIncidentsByJQ(context.Background(), store, mcp.IncidentsInput{
			IncludeOngoing:  &falseVal,
			IncludeResolved: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Single result may be unwrapped or wrapped in array
		var result map[string]any
		switch v := output.Result.(type) {
		case map[string]any:
			result = v
		case []any:
			if len(v) != 1 {
				t.Fatalf("expected 1 incident, got %d", len(v))
			}
			result = v[0].(map[string]any)
		default:
			t.Fatalf("unexpected type: %T", output.Result)
		}
		if result["message"] != "resolved error" {
			t.Errorf("expected 'resolved error', got %v", result["message"])
		}
	})

	t.Run("invalid_jq", func(t *testing.T) {
		_, err := mcp.FetchIncidentsByJQ(context.Background(), store, mcp.IncidentsInput{
			JQ: "invalid{{",
		})
		if err == nil {
			t.Error("expected error for invalid jq query")
		}
	})
}

func TestFetchLogsByJQ(t *testing.T) {
	target, _ := api.ParseURL("https://example.com")

	store := &mockStore{
		name: "test",
		logs: []api.Record{
			{
				Time:    time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
				Status:  api.StatusHealthy,
				Target:  target,
				Latency: 100 * time.Millisecond,
				Message: "ok",
			},
			{
				Time:    time.Date(2021, 1, 2, 3, 4, 6, 0, time.UTC),
				Status:  api.StatusFailure,
				Target:  target,
				Latency: 200 * time.Millisecond,
				Message: "error",
			},
		},
	}

	t.Run("fetch_all", func(t *testing.T) {
		output, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
			Since: "2021-01-01T00:00:00Z",
			Until: "2021-01-03T00:00:00Z",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		results, ok := output.Result.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", output.Result)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 logs, got %d", len(results))
		}
	})

	t.Run("filter_with_jq", func(t *testing.T) {
		output, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
			Since: "2021-01-01T00:00:00Z",
			Until: "2021-01-03T00:00:00Z",
			JQ:    `.[] | select(.status == "HEALTHY")`,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, ok := output.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", output.Result)
		}
		if result["status"] != "HEALTHY" {
			t.Errorf("expected HEALTHY status, got %v", result["status"])
		}
	})

	t.Run("missing_since", func(t *testing.T) {
		_, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
			Until: "2021-01-03T00:00:00Z",
		})
		if err == nil {
			t.Error("expected error for missing since")
		}
	})

	t.Run("missing_until", func(t *testing.T) {
		_, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
			Since: "2021-01-01T00:00:00Z",
		})
		if err == nil {
			t.Error("expected error for missing until")
		}
	})

	t.Run("invalid_since_format", func(t *testing.T) {
		_, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
			Since: "invalid",
			Until: "2021-01-03T00:00:00Z",
		})
		if err == nil {
			t.Error("expected error for invalid since format")
		}
	})

	t.Run("invalid_until_format", func(t *testing.T) {
		_, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
			Since: "2021-01-01T00:00:00Z",
			Until: "invalid",
		})
		if err == nil {
			t.Error("expected error for invalid until format")
		}
	})

	t.Run("invalid_jq", func(t *testing.T) {
		_, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
			Since: "2021-01-01T00:00:00Z",
			Until: "2021-01-03T00:00:00Z",
			JQ:    "invalid{{",
		})
		if err == nil {
			t.Error("expected error for invalid jq query")
		}
	})

	t.Run("log_open_error", func(t *testing.T) {
		errorStore := &mockStore{
			logError: errors.New("log open error"),
		}
		_, err := mcp.FetchLogsByJQ(context.Background(), errorStore, mcp.LogsInput{
			Since: "2021-01-01T00:00:00Z",
			Until: "2021-01-03T00:00:00Z",
		})
		if err == nil {
			t.Error("expected error for log open failure")
		}
	})
}

func TestFetchStatusByJQ_EmptyHistory(t *testing.T) {
	store := &mockStore{
		name:         "test",
		probeHistory: []api.ProbeHistory{},
	}

	output, err := mcp.FetchStatusByJQ(context.Background(), store, mcp.StatusInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, ok := output.Result.([]any)
	if !ok {
		// Empty result might be nil
		if output.Result != nil {
			t.Fatalf("expected []any or nil, got %T", output.Result)
		}
		return
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFetchStatusByJQ_NoLatestRecord(t *testing.T) {
	target, _ := api.ParseURL("https://example.com")

	store := &mockStore{
		name: "test",
		probeHistory: []api.ProbeHistory{
			{
				Target:  target,
				Status:  api.StatusUnknown,
				Records: []api.Record{}, // No records
			},
		},
	}

	output, err := mcp.FetchStatusByJQ(context.Background(), store, mcp.StatusInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Single result may be unwrapped or wrapped in array
	var result map[string]any
	switch v := output.Result.(type) {
	case map[string]any:
		result = v
	case []any:
		if len(v) != 1 {
			t.Fatalf("expected 1 status, got %d", len(v))
		}
		result = v[0].(map[string]any)
	default:
		t.Fatalf("unexpected type: %T", output.Result)
	}

	// latest_log should be nil or empty when no records
	latestLog := result["latest_log"]
	if latestLog != nil {
		if m, ok := latestLog.(map[string]any); ok && len(m) > 0 {
			t.Errorf("expected nil or empty latest_log, got %v", result["latest_log"])
		}
	}
}

func TestFetchIncidentsByJQ_EmptyIncidents(t *testing.T) {
	store := &mockStore{
		name:            "test",
		currentIncident: []*api.Incident{},
		incidentHistory: []*api.Incident{},
	}

	output, err := mcp.FetchIncidentsByJQ(context.Background(), store, mcp.IncidentsInput{
		IncludeResolved: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, ok := output.Result.([]any)
	if !ok {
		if output.Result != nil {
			t.Fatalf("expected []any or nil, got %T", output.Result)
		}
		return
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFetchLogsByJQ_EmptyLogs(t *testing.T) {
	store := &mockStore{
		name: "test",
		logs: []api.Record{},
	}

	output, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
		Since: "2021-01-01T00:00:00Z",
		Until: "2021-01-03T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, ok := output.Result.([]any)
	if !ok {
		if output.Result != nil {
			t.Fatalf("expected []any or nil, got %T", output.Result)
		}
		return
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestJQParseURL_InTools(t *testing.T) {
	target, _ := api.ParseURL("https://example.com:8080/path?q=v")

	store := &mockStore{
		name: "test",
		probeHistory: []api.ProbeHistory{
			{
				Target: target,
				Status: api.StatusHealthy,
				Records: []api.Record{
					{
						Time:    time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
						Status:  api.StatusHealthy,
						Target:  target,
						Latency: 100 * time.Millisecond,
						Message: "ok",
					},
				},
			},
		},
	}

	output, err := mcp.FetchStatusByJQ(context.Background(), store, mcp.StatusInput{
		JQ: `.[0].target | parse_url | .hostname`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Result != "example.com" {
		t.Errorf("expected 'example.com', got %v", output.Result)
	}
}

func TestStatusInput_Sorting(t *testing.T) {
	target1, _ := api.ParseURL("https://zebra.com")
	target2, _ := api.ParseURL("https://alpha.com")

	store := &mockStore{
		name: "test",
		probeHistory: []api.ProbeHistory{
			{
				Target:  target1,
				Status:  api.StatusHealthy,
				Records: []api.Record{},
			},
			{
				Target:  target2,
				Status:  api.StatusHealthy,
				Records: []api.Record{},
			},
		},
	}

	output, err := mcp.FetchStatusByJQ(context.Background(), store, mcp.StatusInput{
		JQ: `.[0].target`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Results should be sorted alphabetically by target
	if output.Result != "https://alpha.com" {
		t.Errorf("expected 'https://alpha.com' (sorted first), got %v", output.Result)
	}
}

func TestIncidentsInput_Sorting(t *testing.T) {
	target, _ := api.ParseURL("https://example.com")

	store := &mockStore{
		name: "test",
		incidentHistory: []*api.Incident{
			{
				Target:   target,
				Status:   api.StatusFailure,
				Message:  "later",
				StartsAt: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
				EndsAt:   time.Date(2021, 2, 1, 1, 0, 0, 0, time.UTC),
			},
			{
				Target:   target,
				Status:   api.StatusFailure,
				Message:  "earlier",
				StartsAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndsAt:   time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
			},
		},
	}

	falseVal := false
	output, err := mcp.FetchIncidentsByJQ(context.Background(), store, mcp.IncidentsInput{
		IncludeOngoing:  &falseVal,
		IncludeResolved: true,
		JQ:              `.[0].message`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Results should be sorted by starts_at_unix
	if output.Result != "earlier" {
		t.Errorf("expected 'earlier' (sorted first by time), got %v", output.Result)
	}
}

func TestOutput_Structure(t *testing.T) {
	output := mcp.Output{
		Result: "test value",
	}

	if output.Result != "test value" {
		t.Errorf("expected 'test value', got %v", output.Result)
	}

	// Test with complex result
	output2 := mcp.Output{
		Result: map[string]any{
			"key": "value",
			"num": 42,
		},
	}

	result, ok := output2.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", output2.Result)
	}
	if diff := cmp.Diff("value", result["key"]); diff != "" {
		t.Errorf("unexpected key value (-want +got):\n%s", diff)
	}
}

func TestFetchLogsByJQ_WithSearch(t *testing.T) {
	target, _ := api.ParseURL("https://example.com")

	store := &mockStore{
		name: "test",
		logs: []api.Record{
			{
				Time:    time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
				Status:  api.StatusHealthy,
				Target:  target,
				Latency: 100 * time.Millisecond,
				Message: "ok",
			},
			{
				Time:    time.Date(2021, 1, 2, 3, 4, 6, 0, time.UTC),
				Status:  api.StatusFailure,
				Target:  target,
				Latency: 200 * time.Millisecond,
				Message: "error",
			},
			{
				Time:    time.Date(2021, 1, 2, 3, 4, 7, 0, time.UTC),
				Status:  api.StatusHealthy,
				Target:  target,
				Latency: 50 * time.Millisecond,
				Message: "recovered",
			},
		},
	}

	t.Run("filter_by_status", func(t *testing.T) {
		output, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
			Since:  "2021-01-01T00:00:00Z",
			Until:  "2021-01-03T00:00:00Z",
			Search: "status=HEALTHY",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		results, ok := output.Result.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", output.Result)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 HEALTHY logs, got %d", len(results))
		}
		for _, r := range results {
			rec := r.(map[string]any)
			if rec["status"] != "HEALTHY" {
				t.Errorf("expected HEALTHY status, got %v", rec["status"])
			}
		}
	})

	t.Run("filter_by_status_failure", func(t *testing.T) {
		output, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
			Since:  "2021-01-01T00:00:00Z",
			Until:  "2021-01-03T00:00:00Z",
			Search: "status=FAILURE",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		results, ok := output.Result.([]any)
		if !ok {
			// Single result may be unwrapped
			if m, ok := output.Result.(map[string]any); ok {
				if m["status"] != "FAILURE" {
					t.Errorf("expected FAILURE status, got %v", m["status"])
				}
				return
			}
			t.Fatalf("expected []any or map[string]any, got %T", output.Result)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 FAILURE log, got %d", len(results))
		}
	})

	t.Run("search_with_jq", func(t *testing.T) {
		output, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
			Since:  "2021-01-01T00:00:00Z",
			Until:  "2021-01-03T00:00:00Z",
			Search: "status=HEALTHY",
			JQ:     `map(.message)`,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		results, ok := output.Result.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", output.Result)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 messages, got %d", len(results))
		}
	})

	t.Run("filter_no_match", func(t *testing.T) {
		output, err := mcp.FetchLogsByJQ(context.Background(), store, mcp.LogsInput{
			Since:  "2021-01-01T00:00:00Z",
			Until:  "2021-01-03T00:00:00Z",
			Search: "status=UNKNOWN",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		results, ok := output.Result.([]any)
		if !ok {
			if output.Result != nil {
				t.Fatalf("expected []any or nil, got %T", output.Result)
			}
			return
		}
		if len(results) != 0 {
			t.Errorf("expected 0 logs, got %d", len(results))
		}
	})
}
