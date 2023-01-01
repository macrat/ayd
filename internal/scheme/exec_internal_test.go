package scheme

import (
	"math"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	api "github.com/macrat/ayd/lib-ayd"
)

func Test_parseExecMessage(t *testing.T) {
	latencyOmit := 42 * time.Hour

	tests := []struct {
		Input         string
		DefaultStatus api.Status
		Output        string
		Status        api.Status
		Latency       time.Duration
		Extra         map[string]any
	}{
		{
			"::foo::bar\n::latency::12.3\nhello\n",
			api.StatusHealthy,
			"hello",
			api.StatusHealthy,
			12300 * time.Microsecond,
			map[string]any{"foo": "bar"},
		},
		{
			"::status::failure\n::hello:: \" world \" ",
			api.StatusHealthy,
			"",
			api.StatusFailure,
			latencyOmit,
			map[string]any{"hello": " world "},
		},
		{
			"A\n::abc::123.456\nB\n::def::ghi\nC\n::status:: Aborted\n",
			api.StatusFailure,
			"A\nB\nC",
			api.StatusAborted,
			latencyOmit,
			map[string]any{"abc": 123.456, "def": "ghi"},
		},
		{
			"::status::abc",
			api.StatusDegrade,
			"",
			api.StatusUnknown,
			latencyOmit,
			map[string]any{},
		},
		{
			"wah\n::latency:: 123",
			api.StatusFailure,
			"wah",
			api.StatusFailure,
			123 * time.Millisecond,
			map[string]any{},
		},
		{
			"hello world\n::latency:: abc",
			api.StatusAborted,
			"hello world",
			api.StatusAborted,
			latencyOmit,
			map[string]any{},
		},
		{
			"hello ::world::",
			api.StatusHealthy,
			"hello ::world::",
			api.StatusHealthy,
			latencyOmit,
			map[string]any{},
		},
		{
			"\n::empty::  \t \n",
			api.StatusHealthy,
			"",
			api.StatusHealthy,
			latencyOmit,
			map[string]any{"empty": ""},
		},
		{
			"::latency:: 12345678901234567890",
			api.StatusHealthy,
			"",
			api.StatusHealthy,
			time.Duration(math.MaxInt64),
			map[string]any{},
		},
	}

	for i, tt := range tests {
		message, status, latency, extra := parseExecMessage(tt.Input, tt.DefaultStatus, latencyOmit)

		if diff := cmp.Diff(tt.Output, message); diff != "" {
			t.Errorf("%d: unexpected message\n%s", i, diff)
		}

		if status != tt.Status {
			t.Errorf("%d: unexpected status: expected %s but got %s", i, tt.Status, status)
		}

		if latency != tt.Latency {
			t.Errorf("%d: unexpected latency: expected %s but got %s", i, tt.Latency, latency)
		}

		if diff := cmp.Diff(tt.Extra, extra); diff != "" {
			t.Errorf("%d: unexpected extra:\n%s", i, diff)
		}
	}
}

func Fuzz_parseExecMessage(f *testing.F) {
	f.Add("hello world\n")
	f.Add("hello\nworld")
	f.Add("::hello::world")
	f.Add("hello\n::latency:: 123.4\nworld\n")
	f.Add("foo\n::status:: degrade\n")
	f.Add("::status::unknown\t\n::latency::12345678901234567890")
	f.Add("a\n::foo::bar\n\nb\n::abc123::{\"hello\":\"world\"}\nc\n::latency::-1\n\nwah")

	f.Fuzz(func(t *testing.T, message string) {
		_, status, latency, extra := parseExecMessage(message, api.StatusHealthy, 42*time.Millisecond)

		if status != api.StatusHealthy && status != api.StatusDegrade && status != api.StatusFailure && status != api.StatusAborted && status != api.StatusUnknown {
			t.Errorf("Invalid status generated: %s", status)
		}

		if latency < 0 {
			t.Errorf("Invalid latency generated: %s", latency)
		}

		if extra == nil {
			t.Errorf("The extra is nil")
		}
	})
}
