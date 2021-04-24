package probe

import (
	"testing"
	"time"

	"github.com/macrat/ayd/store"
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
		Default store.Status
		Output  string
		Status  store.Status
	}{
		{"::status::healthy", store.STATUS_UNKNOWN, "", store.STATUS_HEALTHY},
		{"::status::Failure", store.STATUS_UNKNOWN, "", store.STATUS_FAILURE},
		{"::status::UNKNOWN", store.STATUS_HEALTHY, "", store.STATUS_UNKNOWN},
		{"::status::abcdefg", store.STATUS_UNKNOWN, "::status::abcdefg", store.STATUS_UNKNOWN},
		{"hello\n::status::FAILURE\nworld", store.STATUS_UNKNOWN, "hello\nworld", store.STATUS_FAILURE},
		{"abc\n::status::healthy\n::status::failure", store.STATUS_UNKNOWN, "abc", store.STATUS_FAILURE},
		{"hello\nworld", store.STATUS_UNKNOWN, "hello\nworld", store.STATUS_UNKNOWN},
		{"", store.STATUS_UNKNOWN, "", store.STATUS_UNKNOWN},
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
