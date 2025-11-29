package mcp_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/mcp"
	api "github.com/macrat/ayd/lib-ayd"
)

// mockReporter implements scheme.Reporter for testing.
type mockReporter struct {
	records []api.Record
}

func (r *mockReporter) Report(source *api.URL, rec api.Record) {
	r.records = append(r.records, rec)
}

func (r *mockReporter) DeactivateTarget(source *api.URL, targets ...*api.URL) {}

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &mockReporter{}
	scheduler := mcp.NewScheduler(ctx, reporter)
	defer scheduler.Stop()

	t.Run("success", func(t *testing.T) {
		output, err := mcp.StartMonitoringFunc(scheduler, mcp.StartMonitoringInput{
			Schedule: "5m",
			Targets:  []string{"dummy:healthy"},
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
			Targets: []string{"dummy:healthy"},
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
			Targets:  []string{"dummy:healthy"},
		})
		if err == nil {
			t.Error("expected error for invalid schedule")
		}
	})

	t.Run("invalid_target", func(t *testing.T) {
		_, err := mcp.StartMonitoringFunc(scheduler, mcp.StartMonitoringInput{
			Schedule: "5m",
			Targets:  []string{"invalid-scheme://example.com"},
		})
		if err == nil {
			t.Error("expected error for invalid target")
		}
	})
}

func TestListMonitoring(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &mockReporter{}
	scheduler := mcp.NewScheduler(ctx, reporter)
	defer scheduler.Stop()

	// Start some monitoring entries
	_, err := scheduler.StartMonitoring("5m", []string{"dummy:healthy"})
	if err != nil {
		t.Fatalf("failed to start monitoring: %v", err)
	}
	_, err = scheduler.StartMonitoring("1h", []string{"https://example.org", "https://example.net"})
	if err != nil {
		t.Fatalf("failed to start monitoring: %v", err)
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
	})

	t.Run("filter_no_match", func(t *testing.T) {
		output, err := mcp.ListMonitoringFunc(scheduler, mcp.ListMonitoringInput{
			Keywords: []string{"notfound"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(output.Entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(output.Entries))
		}
	})
}

func TestStopMonitoring(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		reporter := &mockReporter{}
		scheduler := mcp.NewScheduler(ctx, reporter)
		defer scheduler.Stop()

		id, err := scheduler.StartMonitoring("5m", []string{"dummy:healthy"})
		if err != nil {
			t.Fatalf("failed to start monitoring: %v", err)
		}

		output, err := mcp.StopMonitoringFunc(scheduler, mcp.StopMonitoringInput{
			IDs: []string{id},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if diff := cmp.Diff([]string{id}, output.Stopped); diff != "" {
			t.Errorf("stopped mismatch (-want +got):\n%s", diff)
		}
		if len(output.Errors) != 0 {
			t.Errorf("expected no errors, got %v", output.Errors)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		reporter := &mockReporter{}
		scheduler := mcp.NewScheduler(ctx, reporter)
		defer scheduler.Stop()

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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		reporter := &mockReporter{}
		scheduler := mcp.NewScheduler(ctx, reporter)
		defer scheduler.Stop()

		_, err := mcp.StopMonitoringFunc(scheduler, mcp.StopMonitoringInput{
			IDs: []string{},
		})
		if err == nil {
			t.Error("expected error for empty IDs")
		}
	})
}
