package mcp

import (
	"context"
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestDefaultProber_Probe(t *testing.T) {
	prober := NewDefaultProber()

	t.Run("healthy_target", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		rec := prober.Probe(ctx, "dummy:healthy")
		if rec.Status != api.StatusHealthy {
			t.Errorf("expected status HEALTHY, got %s", rec.Status)
		}
	})

	t.Run("failure_target", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		rec := prober.Probe(ctx, "dummy:failure")
		if rec.Status != api.StatusFailure {
			t.Errorf("expected status FAILURE, got %s", rec.Status)
		}
	})

	t.Run("invalid_target", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		rec := prober.Probe(ctx, "invalid-scheme://example.com")
		if rec.Status != api.StatusUnknown {
			t.Errorf("expected status UNKNOWN for invalid target, got %s", rec.Status)
		}
		if rec.Message == "" {
			t.Error("expected error message for invalid target")
		}
	})
}

func TestSingleRecordReporter(t *testing.T) {
	reporter := &singleRecordReporter{}

	target1, _ := api.ParseURL("https://example.com")
	target2, _ := api.ParseURL("https://example.org")

	// First report should be stored
	rec1 := api.Record{
		Time:    time.Now(),
		Status:  api.StatusHealthy,
		Target:  target1,
		Message: "first",
	}
	reporter.Report(target1, rec1)

	if reporter.record == nil {
		t.Fatal("expected record to be stored")
	}
	if reporter.record.Message != "first" {
		t.Errorf("expected message 'first', got %s", reporter.record.Message)
	}

	// Second report should be ignored
	rec2 := api.Record{
		Time:    time.Now(),
		Status:  api.StatusFailure,
		Target:  target2,
		Message: "second",
	}
	reporter.Report(target2, rec2)

	if reporter.record.Message != "first" {
		t.Errorf("expected message to remain 'first', got %s", reporter.record.Message)
	}
}

func TestSingleRecordReporter_DeactivateTarget(t *testing.T) {
	reporter := &singleRecordReporter{}
	target, _ := api.ParseURL("https://example.com")

	// DeactivateTarget should be a no-op
	reporter.DeactivateTarget(target, target)

	// Just verify it doesn't panic
	if reporter.record != nil {
		t.Error("expected record to remain nil")
	}
}
