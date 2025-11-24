package ayd_test

import (
	"errors"
	"testing"
	"time"

	"github.com/macrat/ayd/lib-ayd"
)

func TestParseTime_valid(t *testing.T) {
	tests := []string{
		"2000-01-02T00:03:04+00:00",
		"2000-01-02T00:03:04+0000",
		"2000-01-02T00:03:04-00:00",
		"20000102T00:03:04-0000",
		"20000102T00:03:04.897+00:00",
		"20000102T00:03:04.897-00:00",
		"20000102T00:03:04.897010Z",
		"20000102T000304Z",
		"20000102T00:03:04z",
		"2000-01-02T08:48:04+08:45",
		"2000-01-02T08:48:04+0845",
		"2000-01-01T15:18:04-08:45",
		"2000-01-01T15:18:04-0845",
		"2000-01-02T09:03:04+09:00",
		"2000-01-02T09:03:04+0900",
		"2000-01-01T15:03:04-09:00",
		"2000-01-01T15:03:04-0900",
		"2000-01-02T00:03:04.1Z",
		"2000-01-02T00:03:04.12Z",
		"2000-01-02T00:03:04.123Z",
		"2000-01-02T00:03:04.123456789Z",
		"2000-01-02T00:03:04.999999999Z",
		"2000-01-02T090304.897+09:00",
		"2000-01-02T090304.897010+09:00",
		"2000-01-02T090304.897010+0900",
		"2000-01-02t000304Z",
		"2000-01-02t000304z",
		"2000-01-02_00:03:04.897010Z",
		"2000-01-02_00:03:04.897010z",
		"2000-01-02_00:03:04.897z",
		"20000102_00:03:04Z",
		"20000102_00:03:04z",
		"20000102 00:03:04+00:00",
		"20000102 00:03:04-0000",
		"2000-01-02 00:03:04.897-00:00",
		"2000-01-02 00:03:04.897010z",
		"2000-01-02 00:03:04.897Z",
		"2000-01-02 00:03:04Z",
		"2000-01-02 000304z",
		"2000-01-02 090304+09",
		"2000-01-02 09:03:04.897010+09",
		"01/02 00:03:04AM '00 +0000",
	}

	want := time.Date(2000, 1, 2, 0, 3, 4, 0, time.UTC)

	for _, tt := range tests {
		actual, err := ayd.ParseTime(tt)
		if err != nil {
			t.Errorf("failed to parse %q: %s", tt, err)
		} else if want.Unix() != actual.Truncate(time.Second).Unix() {
			t.Errorf("unexpected result from %q: %s", tt, actual)
		}
	}
}

func TestParseTime_omitTime(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{"2000-01-02T15Z", time.Date(2000, 1, 2, 15, 0, 0, 0, time.UTC)},
		{"2000-01-02T1502Z", time.Date(2000, 1, 2, 15, 2, 0, 0, time.UTC)},
		{"2000-01-02T150203Z", time.Date(2000, 1, 2, 15, 2, 3, 0, time.UTC)},
	}

	for _, tt := range tests {
		actual, err := ayd.ParseTime(tt.input)
		if err != nil {
			t.Errorf("failed to parse %q: %s", tt.input, err)
		} else if !tt.want.Equal(actual) {
			t.Errorf("unexpected result from %q: got %s, want %s", tt.input, actual, tt.want)
		}
	}
}

func TestParseTime_invalid(t *testing.T) {
	tests := []string{
		"2000/01/02 00:03:04",
		"2000-01-02T00:03:0",
		"2000-01-02T00:03:",
		"2000-01-02T00:03",
		"2000-01-02T00:0",
		"2000-01-02T00:",
		"2000-01-02T00",
		"2000-01-02T0",
		"2000-01-02T",
		"2000-01-02",
		"2000-01-0",
		"20000102T00:03:0",
		"20000102T00:03:",
		"20000102T00:03",
		"20000102T00:0",
		"20000102T00:",
		"20000102T00",
		"20000102T0",
		"20000102T",
		"20000102",
		"2000010",
		"2000-01-02T00:03:04.1",
		"2000-01-02T00:03:04.123456789",
		"2000-01-02T00:03:04.9999999999",
		"2000-01-02T00:03:04.",
		"2000-01-02T00:03:04.Z",
		"2000-01-02T00:03:04.+09:00",
	}

	for _, tt := range tests {
		_, err := ayd.ParseTime(tt)
		if !errors.Is(err, ayd.ErrInvalidTime) {
			t.Errorf("unexpected error from %q: %s", tt, err)
		}
	}
}

func BenchmarkParseTime_RFC3339(b *testing.B) {
	input := "2000-01-02T00:03:04Z"

	for b.Loop() {
		_, err := ayd.ParseTime(input)
		if err != nil {
			b.Fatalf("failed to parse %q: %s", input, err)
		}
	}
}

func BenchmarkParseTime_OtherFormats(b *testing.B) {
	input := "20000102_000304.897010+0900"

	for b.Loop() {
		_, err := ayd.ParseTime(input)
		if err != nil {
			b.Fatalf("failed to parse %q: %s", input, err)
		}
	}
}

func BenchmarkParseTime_Invalid(b *testing.B) {
	input := "2000/01/02 00:03:04"

	for b.Loop() {
		_, err := ayd.ParseTime(input)
		if !errors.Is(err, ayd.ErrInvalidTime) {
			b.Fatalf("unexpected error from %q: %s", input, err)
		}
	}
}
