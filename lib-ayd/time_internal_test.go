package ayd

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func Test_parse4digits(t *testing.T) {
	tests := []struct {
		s      string
		want   int
		wantOk bool
	}{
		{"1234", 1234, true},
		{"123d", 0, false},
		{"12c4", 0, false},
		{"1b34", 0, false},
		{"a234", 0, false},
		{"abcd", 0, false},
		{"12345", 1234, true},
	}

	for _, tt := range tests {
		got, gotOk := parse4digits(tt.s)
		if got != tt.want || gotOk != tt.wantOk {
			t.Errorf("parse4digits(%q) = (%d, %v), want (%d, %v)", tt.s, got, gotOk, tt.want, tt.wantOk)
		}
	}
}

func Test_parse2digits(t *testing.T) {
	tests := []struct {
		s      string
		pos    int
		want   int
		wantOk bool
	}{
		{"1234", 0, 12, true},
		{"1234", 1, 23, true},
		{"1234", 2, 34, true},
		{"12c4", 0, 12, true},
		{"1b34", 0, 0, false},
	}

	for _, tt := range tests {
		got, gotOk := parse2digits(tt.s, tt.pos)
		if got != tt.want || gotOk != tt.wantOk {
			t.Errorf("parse2digits(%q, %d) = (%d, %v), want (%d, %v)", tt.s, tt.pos, got, gotOk, tt.want, tt.wantOk)
		}
	}
}

func Test_parseDate(t *testing.T) {
	tests := []struct {
		s           string
		wantYear    int
		wantMonth   int
		wantDay     int
		wantHourPos int
		wantOk      bool
	}{
		{"2001-02-03T16:56:07Z", 2001, 2, 3, 11, true},
		{"1999-12-31T16:56:07Z", 1999, 12, 31, 11, true},
		{"2001/02/03T16:56:07Z", 0, 0, 0, 0, false},
		{"20010203T165607Z", 2001, 2, 3, 9, true},
		{"2001020T165607Z", 0, 0, 0, 0, false},
		{"20010203T16Z", 2001, 2, 3, 9, true},
	}

	for _, tt := range tests {
		gotYear, gotMonth, gotDay, hourPos, gotOk := parseDate(tt.s)

		if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay || hourPos != tt.wantHourPos || gotOk != tt.wantOk {
			t.Errorf("parseDate(%q) = (%d, %d, %d, %d, %v), want (%d, %d, %d, %d, %v)",
				tt.s, gotYear, gotMonth, gotDay, hourPos, gotOk,
				tt.wantYear, tt.wantMonth, tt.wantDay, tt.wantHourPos, tt.wantOk)
		}
	}
}

func Test_parseTimeZone(t *testing.T) {
	tests := []struct {
		s           string
		wantLoc     *time.Location
		wantSignPos int
		wantOk      bool
	}{
		{"2006-01-02T15:04:05Z", time.UTC, 19, true},
		{"2006-01-02T15:04:05+09:00", time.FixedZone("", 9*3600), 19, true},
		{"2006-01-02T15:04:05-05:30", time.FixedZone("", -5*3600-30*60), 19, true},
		{"2006-01-02T15:04:05+0900", time.FixedZone("", 9*3600), 19, true},
		{"2006-01-02T15:04:05-0530", time.FixedZone("", -5*3600-30*60), 19, true},
		{"2006-01-02T15:04:05+09", time.FixedZone("", 9*3600), 19, true},
		{"2006-01-02T15:04:05-05", time.FixedZone("", -5*3600), 19, true},
		{"2006-01-02T15:04:05", nil, 0, false},
		{"2006-01-02T15:04:05?09:00", nil, 0, false},
		{"2006-01-02T15:04:05?0900", nil, 0, false},
		{"2006-01-02T15:04:05?09", nil, 0, false},
	}

	for _, tt := range tests {
		gotLoc, gotSignPos, gotOk := parseTimeZone(tt.s)

		if tt.wantOk != gotOk {
			t.Errorf("parseTimeZone(%q) = ok %v, want %v", tt.s, gotOk, tt.wantOk)
		}
		if tt.wantSignPos != gotSignPos {
			t.Errorf("parseTimeZone(%q) = signPos %d, want %d", tt.s, gotSignPos, tt.wantSignPos)
		}
		if tt.wantOk && gotOk {
			wantDT := time.Date(2000, 1, 1, 0, 0, 0, 0, tt.wantLoc)
			gotDT := time.Date(2000, 1, 1, 0, 0, 0, 0, gotLoc)

			if wantDT.UTC().Sub(gotDT.UTC()) != 0 {
				t.Errorf("parseTimeZone(%q) = timezone offset %v, want %v", tt.s, gotDT.Format("-07:00"), wantDT.Format("-07:00"))
			}
		}
	}
}

func Test_parseTime(t *testing.T) {
	type R struct {
		Hour     int
		Minute   int
		Second   int
		Nsec     int
		Accuracy time.Duration
		Ok       bool
	}

	tests := []struct {
		s     string
		start int
		end   int
		want  R
	}{
		{"15:04:05", 0, 8, R{15, 4, 5, 0, time.Second, true}},
		{"150405", 0, 6, R{15, 4, 5, 0, time.Second, true}},
		{"15:04", 0, 5, R{15, 4, 0, 0, time.Minute, true}},
		{"1504", 0, 4, R{15, 4, 0, 0, time.Minute, true}},
		{"15", 0, 2, R{15, 0, 0, 0, time.Hour, true}},
		{"1", 0, 1, R{0, 0, 0, 0, 0, false}},
		{"2001-01-02T15:04:05Z", 11, 19, R{15, 4, 5, 0, time.Second, true}},
		{"15:04:05.123", 0, 12, R{15, 4, 5, 123000000, time.Millisecond, true}},
		{"15:04:05.123456", 0, 15, R{15, 4, 5, 123456000, time.Microsecond, true}},
		{"15:04:05.123456789", 0, 18, R{15, 4, 5, 123456789, time.Nanosecond, true}},
		{"15:04:05.1234567890123", 0, 22, R{15, 4, 5, 123456789, time.Nanosecond, true}},
	}

	for _, tt := range tests {
		var got R
		got.Hour, got.Minute, got.Second, got.Nsec, got.Accuracy, got.Ok = parseTime(tt.s, tt.start, tt.end)
		if diff := cmp.Diff(tt.want, got); diff != "" {
			t.Errorf("parseTime(%q, %d, %d) mismatch (-want +got):\n%s", tt.s, tt.start, tt.end, diff)
		}
	}
}
