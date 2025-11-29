package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/schedule"
	api "github.com/macrat/ayd/lib-ayd"
)

// mockReporter implements scheme.Reporter for testing.
type mockReporter struct {
	records []api.Record
}

func (r *mockReporter) Report(source *api.URL, rec api.Record) {
	r.records = append(r.records, rec)
}

func (r *mockReporter) DeactivateTarget(source *api.URL, targets ...*api.URL) {
	// No-op for testing
}

func TestScheduler_StartMonitoring(t *testing.T) {
	// Set up mock time for schedule tests
	origTime := schedule.CurrentTime
	schedule.CurrentTime = func() time.Time {
		return time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC)
	}
	defer func() { schedule.CurrentTime = origTime }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &mockReporter{}
	scheduler := NewScheduler(ctx, reporter)
	defer scheduler.Stop()

	t.Run("valid_interval_schedule", func(t *testing.T) {
		id, err := scheduler.StartMonitoring("5m", []string{"dummy:healthy"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if id == "" {
			t.Error("expected non-empty ID")
		}

		// Verify the entry was created
		entries := scheduler.ListMonitoring(nil)
		found := false
		for _, e := range entries {
			if e.ID == id {
				found = true
				if e.Schedule != "5m" {
					t.Errorf("expected schedule '5m', got %s", e.Schedule)
				}
				if len(e.Targets) != 1 || e.Targets[0] != "dummy:healthy" {
					t.Errorf("expected targets [dummy:healthy], got %v", e.Targets)
				}
			}
		}
		if !found {
			t.Error("expected to find the scheduled entry")
		}
	})

	t.Run("valid_cron_schedule", func(t *testing.T) {
		id, err := scheduler.StartMonitoring("0 0 * * ?", []string{"dummy:healthy"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if id == "" {
			t.Error("expected non-empty ID")
		}
	})

	t.Run("invalid_schedule", func(t *testing.T) {
		_, err := scheduler.StartMonitoring("invalid", []string{"dummy:healthy"})
		if err == nil {
			t.Error("expected error for invalid schedule")
		}
	})

	t.Run("invalid_target", func(t *testing.T) {
		_, err := scheduler.StartMonitoring("5m", []string{"invalid-scheme://example.com"})
		if err == nil {
			t.Error("expected error for invalid target")
		}
	})
}

func TestScheduler_StopMonitoring(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &mockReporter{}
	scheduler := NewScheduler(ctx, reporter)
	defer scheduler.Stop()

	// Start a monitoring entry
	id, err := scheduler.StartMonitoring("5m", []string{"dummy:healthy"})
	if err != nil {
		t.Fatalf("failed to start monitoring: %v", err)
	}

	t.Run("stop_existing_entry", func(t *testing.T) {
		stopped, errors := scheduler.StopMonitoring([]string{id})
		if len(errors) != 0 {
			t.Errorf("expected no errors, got: %v", errors)
		}
		if len(stopped) != 1 || stopped[0] != id {
			t.Errorf("expected stopped [%s], got %v", id, stopped)
		}

		// Verify entry is removed
		entries := scheduler.ListMonitoring(nil)
		for _, e := range entries {
			if e.ID == id {
				t.Error("entry should have been removed")
			}
		}
	})

	t.Run("stop_nonexistent_entry", func(t *testing.T) {
		stopped, errors := scheduler.StopMonitoring([]string{"nonexistent-id"})
		if len(stopped) != 0 {
			t.Errorf("expected no stopped, got: %v", stopped)
		}
		if len(errors) != 1 {
			t.Errorf("expected one error, got: %v", errors)
		}
	})
}

func TestScheduler_ListMonitoring(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &mockReporter{}
	scheduler := NewScheduler(ctx, reporter)
	defer scheduler.Stop()

	// Start multiple monitoring entries
	_, err := scheduler.StartMonitoring("5m", []string{"dummy:healthy"})
	if err != nil {
		t.Fatalf("failed to start monitoring: %v", err)
	}
	_, err = scheduler.StartMonitoring("10m", []string{"https://example.com", "https://example.org"})
	if err != nil {
		t.Fatalf("failed to start monitoring: %v", err)
	}

	t.Run("list_all", func(t *testing.T) {
		entries := scheduler.ListMonitoring(nil)
		if len(entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(entries))
		}
	})

	t.Run("filter_by_keyword", func(t *testing.T) {
		entries := scheduler.ListMonitoring([]string{"example.com"})
		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}
	})

	t.Run("filter_no_match", func(t *testing.T) {
		entries := scheduler.ListMonitoring([]string{"notfound"})
		if len(entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(entries))
		}
	})
}

func TestMatchKeywords(t *testing.T) {
	tests := []struct {
		name     string
		targets  []string
		keywords []string
		expected bool
	}{
		{
			name:     "empty_keywords",
			targets:  []string{"https://example.com"},
			keywords: []string{},
			expected: true,
		},
		{
			name:     "single_keyword_match",
			targets:  []string{"https://example.com"},
			keywords: []string{"example"},
			expected: true,
		},
		{
			name:     "single_keyword_no_match",
			targets:  []string{"https://example.com"},
			keywords: []string{"notfound"},
			expected: false,
		},
		{
			name:     "multiple_keywords_all_match",
			targets:  []string{"https://example.com", "https://example.org"},
			keywords: []string{"example", "com"},
			expected: true,
		},
		{
			name:     "multiple_keywords_partial_match",
			targets:  []string{"https://example.com"},
			keywords: []string{"example", "org"},
			expected: false,
		},
		{
			name:     "multiple_targets_single_keyword",
			targets:  []string{"https://foo.com", "https://bar.com"},
			keywords: []string{"bar"},
			expected: true,
		},
		{
			name:     "keyword_in_path",
			targets:  []string{"https://example.com/api/v1"},
			keywords: []string{"api"},
			expected: true,
		},
		{
			name:     "empty_targets",
			targets:  []string{},
			keywords: []string{"example"},
			expected: false,
		},
		{
			name:     "empty_both",
			targets:  []string{},
			keywords: []string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchKeywords(tt.targets, tt.keywords)
			if result != tt.expected {
				t.Errorf("matchKeywords(%v, %v) = %v, expected %v",
					tt.targets, tt.keywords, result, tt.expected)
			}
		})
	}
}
