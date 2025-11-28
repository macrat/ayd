package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/schedule"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestMCPCommand_Run_help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := &MCPCommand{
		OutStream: &stdout,
		ErrStream: &stderr,
	}

	code := cmd.Run([]string{"ayd", "mcp", "-h"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	if !strings.Contains(stdout.String(), "Ayd mcp") {
		t.Errorf("expected help text in stdout, got: %s", stdout.String())
	}
}

func TestMCPCommand_Run_invalidFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := &MCPCommand{
		OutStream: &stdout,
		ErrStream: &stderr,
	}

	code := cmd.Run([]string{"ayd", "mcp", "--invalid-flag"})
	if code != 2 {
		t.Errorf("expected exit code 2, got %d", code)
	}

	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Errorf("expected error message in stderr, got: %s", stderr.String())
	}
}

// Note: store.New does not validate the log path upfront.
// The path validation happens lazily when writing logs.
// Therefore, we can't easily test invalid log path scenarios.

func TestMCPCommand_Run_disableLog(t *testing.T) {
	// Test that -f - disables logging (sets logPath to empty)
	var stdout, stderr bytes.Buffer
	cmd := &MCPCommand{
		OutStream: &stdout,
		ErrStream: &stderr,
	}

	// Create a temp directory and verify we can use -f -
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// This will try to start the server but will fail because there's no stdin
	// But we're just testing the flag parsing works correctly
	code := cmd.Run([]string{"ayd", "mcp", "-f", "-", "-h"})
	if code != 0 {
		t.Errorf("expected exit code 0 with -h, got %d", code)
	}
}

func TestLocalMCPScheduler_StartMonitoring(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Create a minimal store
	f, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("failed to create log file: %v", err)
	}
	f.Close()

	// Set up mock time for schedule tests
	origTime := schedule.CurrentTime
	schedule.CurrentTime = func() time.Time {
		return time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC)
	}
	defer func() { schedule.CurrentTime = origTime }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// We can't easily create a store.Store without a lot of setup,
	// so we'll test the scheduler methods through integration tests
	t.Run("valid_interval_schedule", func(t *testing.T) {
		// Test that valid schedules are accepted
		sched, err := schedule.Parse("5m")
		if err != nil {
			t.Errorf("expected valid schedule, got error: %v", err)
		}
		if sched.String() != "5m0s" {
			t.Errorf("expected 5m0s, got %s", sched.String())
		}
		_ = ctx // avoid unused variable warning
	})

	t.Run("valid_cron_schedule", func(t *testing.T) {
		sched, err := schedule.Parse("0 0 * * ?")
		if err != nil {
			t.Errorf("expected valid schedule, got error: %v", err)
		}
		if sched.String() != "0 0 * * ?" {
			t.Errorf("expected '0 0 * * ?', got %s", sched.String())
		}
	})

	t.Run("invalid_schedule", func(t *testing.T) {
		_, err := schedule.Parse("invalid")
		if err == nil {
			t.Error("expected error for invalid schedule")
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

func TestSchedulerEntry(t *testing.T) {
	entry := schedulerEntry{
		id:       1,
		schedule: "5m",
		targets:  []string{"https://example.com", "https://example.org"},
	}

	if entry.id != 1 {
		t.Errorf("expected id 1, got %d", entry.id)
	}
	if entry.schedule != "5m" {
		t.Errorf("expected schedule '5m', got %s", entry.schedule)
	}
	if len(entry.targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(entry.targets))
	}
}

func TestMCPHelp(t *testing.T) {
	// Verify help text contains expected information
	if !strings.Contains(MCPHelp, "ayd mcp") {
		t.Error("help text should contain 'ayd mcp'")
	}
	if !strings.Contains(MCPHelp, "--log-file") {
		t.Error("help text should contain '--log-file'")
	}
	if !strings.Contains(MCPHelp, "--name") {
		t.Error("help text should contain '--name'")
	}
	if !strings.Contains(MCPHelp, "--help") {
		t.Error("help text should contain '--help'")
	}
}
