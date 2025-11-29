package mcp_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/mcp"
)

// mockScheduler is a mock implementation of mcp.Scheduler for testing.
type mockScheduler struct {
	entries []mcp.MonitoringEntry
	nextID  int
}

func (s *mockScheduler) StartMonitoring(schedule string, targets []string) (string, error) {
	if schedule == "invalid" {
		return "", fmt.Errorf("invalid schedule")
	}
	s.nextID++
	id := fmt.Sprintf("monitor-%d", s.nextID)
	s.entries = append(s.entries, mcp.MonitoringEntry{
		ID:       id,
		Schedule: schedule,
		Targets:  targets,
	})
	return id, nil
}

func (s *mockScheduler) StopMonitoring(ids []string) ([]string, []string) {
	var stopped, errors []string
	for _, id := range ids {
		found := false
		for i, entry := range s.entries {
			if entry.ID == id {
				s.entries = append(s.entries[:i], s.entries[i+1:]...)
				stopped = append(stopped, id)
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Sprintf("%s: not found", id))
		}
	}
	return stopped, errors
}

func (s *mockScheduler) ListMonitoring(keywords []string) []mcp.MonitoringEntry {
	if len(keywords) == 0 {
		return s.entries
	}

	var filtered []mcp.MonitoringEntry
	for _, entry := range s.entries {
		match := true
		for _, keyword := range keywords {
			found := false
			for _, target := range entry.Targets {
				if containsKeyword(target, keyword) {
					found = true
					break
				}
			}
			if !found {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func containsKeyword(s, keyword string) bool {
	return len(keyword) == 0 || (len(s) >= len(keyword) && findSubstring(s, keyword))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestCheckTarget(t *testing.T) {
	t.Run("single_target", func(t *testing.T) {
		output, err := mcp.CheckTarget(context.Background(), mcp.CheckTargetInput{
			Targets: []string{"dummy:healthy"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(output.Results) != 1 {
			t.Errorf("expected 1 result, got %d", len(output.Results))
		}
		if output.Results[0]["status"] != "HEALTHY" {
			t.Errorf("expected HEALTHY status, got %v", output.Results[0]["status"])
		}
	})

	t.Run("multiple_targets", func(t *testing.T) {
		output, err := mcp.CheckTarget(context.Background(), mcp.CheckTargetInput{
			Targets: []string{"dummy:healthy", "dummy:failure"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(output.Results) != 2 {
			t.Errorf("expected 2 results, got %d", len(output.Results))
		}

		// Check that we have both healthy and failure results
		var hasHealthy, hasFailure bool
		for _, result := range output.Results {
			if result["status"] == "HEALTHY" {
				hasHealthy = true
			}
			if result["status"] == "FAILURE" {
				hasFailure = true
			}
		}
		if !hasHealthy {
			t.Error("expected a HEALTHY result")
		}
		if !hasFailure {
			t.Error("expected a FAILURE result")
		}
	})

	t.Run("no_targets", func(t *testing.T) {
		_, err := mcp.CheckTarget(context.Background(), mcp.CheckTargetInput{
			Targets: []string{},
		})
		if err == nil {
			t.Error("expected error for empty targets")
		}
	})

	t.Run("invalid_target", func(t *testing.T) {
		output, err := mcp.CheckTarget(context.Background(), mcp.CheckTargetInput{
			Targets: []string{"invalid-scheme://example.com"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(output.Results) != 1 {
			t.Errorf("expected 1 result, got %d", len(output.Results))
		}
		if output.Results[0]["status"] != "UNKNOWN" {
			t.Errorf("expected UNKNOWN status for invalid target, got %v", output.Results[0]["status"])
		}
	})
}

func TestStartMonitoring(t *testing.T) {
	scheduler := &mockScheduler{}

	t.Run("success", func(t *testing.T) {
		output, err := mcp.StartMonitoringFunc(scheduler, mcp.StartMonitoringInput{
			Schedule: "5m",
			Targets:  []string{"https://example.com"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.ID == "" {
			t.Error("expected non-empty ID")
		}
	})

	t.Run("no_schedule", func(t *testing.T) {
		_, err := mcp.StartMonitoringFunc(scheduler, mcp.StartMonitoringInput{
			Targets: []string{"https://example.com"},
		})
		if err == nil {
			t.Error("expected error for empty schedule")
		}
	})

	t.Run("no_targets", func(t *testing.T) {
		_, err := mcp.StartMonitoringFunc(scheduler, mcp.StartMonitoringInput{
			Schedule: "5m",
			Targets:  []string{},
		})
		if err == nil {
			t.Error("expected error for empty targets")
		}
	})

	t.Run("invalid_schedule", func(t *testing.T) {
		_, err := mcp.StartMonitoringFunc(scheduler, mcp.StartMonitoringInput{
			Schedule: "invalid",
			Targets:  []string{"https://example.com"},
		})
		if err == nil {
			t.Error("expected error for invalid schedule")
		}
	})
}

func TestListMonitoring(t *testing.T) {
	scheduler := &mockScheduler{
		entries: []mcp.MonitoringEntry{
			{ID: "1", Schedule: "5m", Targets: []string{"https://example.com"}},
			{ID: "2", Schedule: "1h", Targets: []string{"https://example.org", "https://example.net"}},
		},
	}

	t.Run("all", func(t *testing.T) {
		output, err := mcp.ListMonitoringFunc(scheduler, mcp.ListMonitoringInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(output.Entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(output.Entries))
		}
	})

	t.Run("filter_single_keyword", func(t *testing.T) {
		output, err := mcp.ListMonitoringFunc(scheduler, mcp.ListMonitoringInput{
			Keywords: []string{"example.org"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(output.Entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(output.Entries))
		}
		if output.Entries[0].ID != "2" {
			t.Errorf("expected ID 2, got %s", output.Entries[0].ID)
		}
	})
}

func TestStopMonitoring(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		scheduler := &mockScheduler{
			entries: []mcp.MonitoringEntry{
				{ID: "1", Schedule: "5m", Targets: []string{"https://example.com"}},
				{ID: "2", Schedule: "1h", Targets: []string{"https://example.org"}},
			},
		}

		output, err := mcp.StopMonitoringFunc(scheduler, mcp.StopMonitoringInput{
			IDs: []string{"1"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if diff := cmp.Diff([]string{"1"}, output.Stopped); diff != "" {
			t.Errorf("stopped mismatch (-want +got):\n%s", diff)
		}
		if len(output.Errors) != 0 {
			t.Errorf("expected no errors, got %v", output.Errors)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		scheduler := &mockScheduler{
			entries: []mcp.MonitoringEntry{
				{ID: "1", Schedule: "5m", Targets: []string{"https://example.com"}},
			},
		}

		output, err := mcp.StopMonitoringFunc(scheduler, mcp.StopMonitoringInput{
			IDs: []string{"nonexistent"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(output.Stopped) != 0 {
			t.Errorf("expected no stopped, got %v", output.Stopped)
		}
		if len(output.Errors) != 1 {
			t.Errorf("expected 1 error, got %d", len(output.Errors))
		}
	})

	t.Run("no_ids", func(t *testing.T) {
		scheduler := &mockScheduler{}

		_, err := mcp.StopMonitoringFunc(scheduler, mcp.StopMonitoringInput{
			IDs: []string{},
		})
		if err == nil {
			t.Error("expected error for empty IDs")
		}
	})
}
