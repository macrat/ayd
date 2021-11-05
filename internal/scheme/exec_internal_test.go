package scheme

import (
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestGetLatencyByMessage(t *testing.T) {
	tests := []struct {
		Input   string
		Default time.Duration
		Output  string
		Latency time.Duration
	}{
		{"::latency::123.456", 10 * time.Millisecond, "", 123456 * time.Microsecond},
		{
			"abc\n::latency::\n::latency::123.456\nsomething",
			10 * time.Millisecond,
			"abc\n::latency::\nsomething",
			123456 * time.Microsecond,
		},
		{"::latency::abc\n::latency::654.321\n::latency::123", 10 * time.Millisecond, "::latency::abc", 123 * time.Millisecond},
		{"::latency::\n::latency::a123", 10 * time.Millisecond, "::latency::\n::latency::a123", 10 * time.Millisecond},
		{"", 10 * time.Millisecond, "", 10 * time.Millisecond},
	}

	for _, tt := range tests {
		message, latency := getLatencyByMessage(tt.Input, tt.Default)

		if message != tt.Output {
			t.Errorf("unexpected message\nexpected: %#v\n but got: %#v", tt.Output, message)
		}
		if latency != tt.Latency {
			t.Errorf("unexpected latency\nexpected: %s\n but got: %s", tt.Latency, latency)
		}
	}
}

func TestGetStatusByMessage(t *testing.T) {
	tests := []struct {
		Input   string
		Default api.Status
		Output  string
		Status  api.Status
	}{
		{"::status::healthy", api.StatusUnknown, "", api.StatusHealthy},
		{"::status::Failure", api.StatusUnknown, "", api.StatusFailure},
		{"::status::aborted", api.StatusUnknown, "", api.StatusAborted},
		{"::status::UNKNOWN", api.StatusHealthy, "", api.StatusUnknown},
		{"::status::abcdefg", api.StatusUnknown, "::status::abcdefg", api.StatusUnknown},
		{"hello\n::status::FAILURE\nworld", api.StatusUnknown, "hello\nworld", api.StatusFailure},
		{"abc\n::status::healthy\n::status::failure", api.StatusUnknown, "abc", api.StatusFailure},
		{"hello\nworld", api.StatusUnknown, "hello\nworld", api.StatusUnknown},
		{"", api.StatusUnknown, "", api.StatusUnknown},
	}

	for _, tt := range tests {
		message, status := getStatusByMessage(tt.Input, tt.Default)

		if message != tt.Output {
			t.Errorf("unexpected message\nexpected: %#v\n but got: %#v", tt.Output, message)
		}
		if status != tt.Status {
			t.Errorf("unexpected status\nexpected: %s\n but got: %s", tt.Status, status)
		}
	}
}
