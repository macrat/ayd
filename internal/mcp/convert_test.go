package mcp_test

import (
	"testing"
	"time"

	"github.com/macrat/ayd/internal/mcp"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestRecordToMap(t *testing.T) {
	target, _ := api.ParseURL("https://example.com")
	rec := api.Record{
		Time:    time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
		Status:  api.StatusHealthy,
		Latency: 123 * time.Millisecond,
		Target:  target,
		Message: "hello",
		Extra:   map[string]any{"extra": "value"},
	}

	m := mcp.RecordToMap(rec)

	if m["time"] != "2021-01-02T03:04:05Z" {
		t.Errorf("unexpected time: %v", m["time"])
	}
	if m["time_unix"].(int64) != 1609556645 {
		t.Errorf("unexpected time_unix: %v", m["time_unix"])
	}
	if m["status"] != "HEALTHY" {
		t.Errorf("unexpected status: %v", m["status"])
	}
	if m["latency"] != "123ms" {
		t.Errorf("unexpected latency: %v", m["latency"])
	}
	if m["latency_ms"].(float64) != 123.0 {
		t.Errorf("unexpected latency_ms: %v", m["latency_ms"])
	}
	if m["target"] != "https://example.com" {
		t.Errorf("unexpected target: %v", m["target"])
	}
	if m["message"] != "hello" {
		t.Errorf("unexpected message: %v", m["message"])
	}
	if m["extra"] != "value" {
		t.Errorf("unexpected extra: %v", m["extra"])
	}
}

func TestIncidentToMap(t *testing.T) {
	target, _ := api.ParseURL("https://example.com")

	t.Run("ongoing", func(t *testing.T) {
		inc := &api.Incident{
			Target:   target,
			Status:   api.StatusFailure,
			Message:  "error",
			StartsAt: time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
		}

		m := mcp.IncidentToMap(inc)

		if m["target"] != "https://example.com" {
			t.Errorf("unexpected target: %v", m["target"])
		}
		if m["status"] != "FAILURE" {
			t.Errorf("unexpected status: %v", m["status"])
		}
		if m["message"] != "error" {
			t.Errorf("unexpected message: %v", m["message"])
		}
		if m["starts_at"] != "2021-01-02T03:04:05Z" {
			t.Errorf("unexpected starts_at: %v", m["starts_at"])
		}
		if m["starts_at_unix"].(int64) != 1609556645 {
			t.Errorf("unexpected starts_at_unix: %v", m["starts_at_unix"])
		}
		if m["ends_at"] != nil {
			t.Errorf("unexpected ends_at: %v", m["ends_at"])
		}
		if m["ends_at_unix"] != nil {
			t.Errorf("unexpected ends_at_unix: %v", m["ends_at_unix"])
		}
	})

	t.Run("resolved", func(t *testing.T) {
		inc := &api.Incident{
			Target:   target,
			Status:   api.StatusFailure,
			Message:  "error",
			StartsAt: time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
			EndsAt:   time.Date(2021, 1, 2, 4, 5, 6, 0, time.UTC),
		}

		m := mcp.IncidentToMap(inc)

		if m["ends_at"] != "2021-01-02T04:05:06Z" {
			t.Errorf("unexpected ends_at: %v", m["ends_at"])
		}
		if m["ends_at_unix"].(int64) != 1609560306 {
			t.Errorf("unexpected ends_at_unix: %v", m["ends_at_unix"])
		}
	})
}
