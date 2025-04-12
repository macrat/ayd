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
		"2000-01-02T09:03:04+09:00",
		"2000-01-02T09:03:04+0900",
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

func TestParseTime_invalid(t *testing.T) {
	tests := []string{
		"2000/01/02 00:03:04",
	}

	for _, tt := range tests {
		_, err := ayd.ParseTime(tt)
		if !errors.Is(err, ayd.ErrInvalidTime) {
			t.Errorf("unexpected error from %q: %s", tt, err)
		}
	}
}
