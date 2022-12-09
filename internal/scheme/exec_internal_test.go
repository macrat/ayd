package scheme

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
		{"::latency::\t0.123", 10 * time.Millisecond, "", 123 * time.Microsecond},
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
		{"::status::DeGrade", api.StatusUnknown, "", api.StatusDegrade},
		{"::status::Failure", api.StatusUnknown, "", api.StatusFailure},
		{"::status::aborted ", api.StatusUnknown, "", api.StatusAborted},
		{"::status:: UNKNOWN", api.StatusHealthy, "", api.StatusUnknown},
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

func TestGetExtraByMessage(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
		Extra  map[string]any
	}{
		{`::foo::bar`, "", map[string]any{"foo": "bar"}},
		{`::hello:: " world " `, "", map[string]any{"hello": " world "}},
		{"A\n::abc::123.456\nB\n::def::ghi\nC", "A\nB\nC", map[string]any{"abc": 123.456, "def": "ghi"}},
		{"::status::abc", "::status::abc", map[string]any{}},
		{"::latency:: 123", "::latency:: 123", map[string]any{}},
		{"hello world", "hello world", map[string]any{}},
		{"hello ::world::", "hello ::world::", map[string]any{}},
		{"\n::empty::  \t \n", "", map[string]any{"empty": ""}},
	}

	for _, tt := range tests {
		message, extra := getExtraByMessage(tt.Input)

		if message != tt.Output {
			t.Errorf("unexpected message\nexpected: %#v\n but got: %#v", tt.Output, message)
		}
		if diff := cmp.Diff(tt.Extra, extra); diff != "" {
			t.Errorf("unexpected extra:\n%s", diff)
		}
	}
}
