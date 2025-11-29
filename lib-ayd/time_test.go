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
		"01/02 00:03:04AM '00 -0000",
		"Mon Jan  2 00:03:04 2000",
		"Mon Jan  2 00:03:04 UTC 2000",
		"Mon Jan 02 00:03:04 +0000 2000",
		"Mon Jan 02 08:33:04 +0830 2000",
		"Mon Jan 01 23:03:04 -0100 2000",
		"Monday, 02-Jan-00 00:03:04 UTC",
		"Mon, 02 Jan 2000 00:03:04 UTC",
		"Mon, 02 Jan 2000 00:03:04 +0000",
		"Mon, 02 Jan 2000 00:03:04 -0000",
		"Mon, 02 Jan 2000 09:03:04 +0900",
		"Mon, 01 Jan 2000 22:03:04 -0200",
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
		{"02 Jan 00 00:03 UTC", time.Date(2000, 1, 2, 0, 3, 0, 0, time.UTC)},
		{"02 Jan 00 00:03 +0000", time.Date(2000, 1, 2, 0, 3, 0, 0, time.UTC)},
		{"02 Jan 00 00:03 -0000", time.Date(2000, 1, 2, 0, 3, 0, 0, time.UTC)},
		{"02 Jan 00 09:03 +0900", time.Date(2000, 1, 2, 0, 3, 0, 0, time.UTC)},
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
		"2000-01-02T00:03:04.12!345Z",
	}

	for _, tt := range tests {
		_, err := ayd.ParseTime(tt)
		if !errors.Is(err, ayd.ErrInvalidTime) {
			t.Errorf("unexpected error from %q: %s", tt, err)
		}
	}
}

func TestParseTime_WithLocation(t *testing.T) {
	jst := time.FixedZone("JST", 9*3600)

	tests := []struct {
		input      string
		location   *time.Location
		want       time.Time
		resolution time.Duration
	}{
		{"2000-01-02 03:45:06", jst, time.Date(2000, 1, 2, 3, 45, 6, 0, jst), time.Second},
		{"2000-01-02T15:04", jst, time.Date(2000, 1, 2, 15, 4, 0, 0, jst), time.Minute},
		{"2001-02-03T16", time.UTC, time.Date(2001, 2, 3, 16, 0, 0, 0, time.UTC), time.Hour},
	}

	for _, tt := range tests {
		actual, res, err := ayd.ParseTimeWithResolution(tt.input, tt.location)
		if err != nil {
			t.Errorf("failed to parse %q: %s", tt.input, err)
		} else if !tt.want.Equal(actual) {
			t.Errorf("unexpected result from %q: got %s, want %s", tt.input, actual, tt.want)
		} else if tt.resolution != res {
			t.Errorf("unexpected resolution from %q: got %s, want %s", tt.input, res, tt.resolution)
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

func FuzzParseTime(f *testing.F) {
	f.Add("2000-01-02T00:03:04Z")
	f.Add("2000-01-02T15:04:05+09:00")
	f.Add("2000-01-02 03:45:06+01:23")
	f.Add("2000-01-02_03:45:06-0200")
	f.Add("2000102 030405Z")
	f.Add("Mon Jan 02 15:04:05 MST 2006")
	f.Add("Mon, 02 Jan 2006 15:04:05 -0700")
	f.Add("Monday, 02-Jan-06 15:04:05 MST")
	f.Add("02 Jan 06 15:04 MST")

	f.Fuzz(func(t *testing.T, input string) {
		_, err := ayd.ParseTime(input)
		if err != nil && !errors.Is(err, ayd.ErrInvalidTime) {
			t.Errorf("unexpected error for input %q: %s", input, err)
		}
	})
}
