package ayd

import (
	"errors"
	"time"
)

var (
	ErrInvalidTime = errors.New("invalid format")
)

func isNotDigit(c byte) bool {
	return c < '0' || '9' < c
}

// parse4digits parses 4 digit int number from the beginning of s.
// Returns (0, false) if s is not digits.
//
// IMPORTANT: s must have at least 4 characters. This function does not check the length of s.
func parse4digits(s string) (int, bool) {
	if isNotDigit(s[0]) || isNotDigit(s[1]) || isNotDigit(s[2]) || isNotDigit(s[3]) {
		return 0, false
	}
	d1 := s[0] - '0'
	d2 := s[1] - '0'
	d3 := s[2] - '0'
	d4 := s[3] - '0'
	return int(d1)*1000 + int(d2)*100 + int(d3)*10 + int(d4), true
}

// parse2digits parses 2 digit int number from s starting at pos.
// Returns (0, false) if s[pos:pos+2] is not digits.
//
// IMPORTANT: s must have at least pos+2 characters. This function does not check the length of s.
func parse2digits(s string, pos int) (int, bool) {
	if isNotDigit(s[pos]) || isNotDigit(s[pos+1]) {
		return 0, false
	}
	d1 := s[pos] - '0'
	d2 := s[pos+1] - '0'
	return int(d1)*10 + int(d2), true
}

func isDateTimeDelim(c byte) bool {
	return c == 'T' || c == 't' || c == ' ' || c == '_'
}

func parseDate(s string) (year, month, day, hourPos int, ok bool) {
	var monthPos, dayPos int

	if len(s) < len("20060102T15Z") {
		// Too short to be valid RFC3339 format
		return 0, 0, 0, 0, false
	} else if isDateTimeDelim(s[8]) {
		// Format: 20060102T...
		monthPos = 4
		dayPos = 6
		hourPos = 9
	} else if isDateTimeDelim(s[10]) && s[4] == '-' && s[7] == '-' {
		// Format: 2006-01-02T...
		monthPos = 5
		dayPos = 8
		hourPos = 11
	} else {
		// Invalid date part
		return 0, 0, 0, 0, false
	}

	year, yok := parse4digits(s)
	month, mok := parse2digits(s, monthPos)
	day, dok := parse2digits(s, dayPos)

	return year, month, day, hourPos, yok && mok && dok
}

func parseTimeZone(s string) (tz *time.Location, signPos int, ok bool) {
	slen := len(s)

	if s[slen-1] == 'Z' || s[slen-1] == 'z' {
		return time.UTC, slen - 1, true
	}

	var h, m int
	var hok, mok bool

	switch s[slen-3] {
	case ':':
		// Format: ...+07:00
		h, hok = parse2digits(s, slen-5)
		m, mok = parse2digits(s, slen-2)
		signPos = slen - 6
	case '+', '-':
		// Format: ...+07
		h, hok = parse2digits(s, slen-2)
		mok = true
		signPos = slen - 3
	default:
		// Format: ...+0700
		h, hok = parse2digits(s, slen-4)
		m, mok = parse2digits(s, slen-2)
		signPos = slen - 5
	}

	if !hok || !mok {
		return nil, 0, false
	}

	switch s[signPos] {
	case '+':
		offset := h*3600 + m*60
		return time.FixedZone("", offset), signPos, true
	case '-':
		offset := -h*3600 - m*60
		return time.FixedZone("", offset), signPos, true
	default:
		return nil, 0, false
	}
}

func parseTime(s string, start, end int) (hour, minute, second, nsec int, resolution time.Duration, ok bool) {
	switch end - start {
	case len("15:04:05"):
		hour, hok := parse2digits(s, start)
		minute, mok := parse2digits(s, start+3)
		second, sok := parse2digits(s, start+6)
		return hour, minute, second, 0, time.Second, hok && mok && sok
	case len("150405"):
		hour, hok := parse2digits(s, start)
		minute, mok := parse2digits(s, start+2)
		second, sok := parse2digits(s, start+4)
		return hour, minute, second, 0, time.Second, hok && mok && sok
	case len("15:04"):
		hour, hok := parse2digits(s, start)
		minute, mok := parse2digits(s, start+3)
		return hour, minute, 0, 0, time.Minute, hok && mok
	case len("1504"):
		hour, hok := parse2digits(s, start)
		minute, mok := parse2digits(s, start+2)
		return hour, minute, 0, 0, time.Minute, hok && mok
	case len("15"):
		hour, hok := parse2digits(s, start)
		return hour, 0, 0, 0, time.Hour, hok
	}

	var hok, mok, sok bool
	nsecStart := end

	if end-start > len("150405.") && s[start+6] == '.' {
		hour, hok = parse2digits(s, start)
		minute, mok = parse2digits(s, start+2)
		second, sok = parse2digits(s, start+4)
		nsecStart = start + 7
	} else if end-start > len("15:04:05.") && s[start+8] == '.' {
		hour, hok = parse2digits(s, start)
		minute, mok = parse2digits(s, start+3)
		second, sok = parse2digits(s, start+6)
		nsecStart = start + 9
	} else {
		return 0, 0, 0, 0, 0, false
	}

	resolution = time.Second
	for i := range 9 {
		var d int
		if nsecStart+i < end {
			if isNotDigit(s[nsecStart+i]) {
				return 0, 0, 0, 0, 0, false
			}
			d = int(s[nsecStart+i] - '0')
			resolution /= 10
		}
		nsec = nsec*10 + d
	}

	return hour, minute, second, nsec, resolution, hok && mok && sok
}

func parseRFC3339(s string, defaultTimezone *time.Location) (time.Time, time.Duration, bool) {
	year, month, day, hourPos, ok := parseDate(s)
	if !ok {
		return time.Time{}, 0, false
	}

	tz, signPos, ok := parseTimeZone(s)
	if !ok {
		if defaultTimezone == nil {
			return time.Time{}, 0, false
		} else {
			tz = defaultTimezone
			signPos = len(s)
		}
	}

	hour, minute, second, nsec, resolution, ok := parseTime(s, hourPos, signPos)
	if !ok {
		return time.Time{}, 0, false
	}

	return time.Date(year, time.Month(month), day, hour, minute, second, nsec, tz), resolution, true
}

// ParseTimeWithResolution parses time string in Ayd way and returns time and its resolution.
// This function supports RFC3339 and some variant formats.
//
// If time zone is not specified in the string, the provided tz location is used.
// If tz is nil, it returns ErrInvalidTime for time strings without time zone.
func ParseTimeWithResolution(s string, tz *time.Location) (time.Time, time.Duration, error) {
	if t, acc, ok := parseRFC3339(s, tz); ok {
		return t, acc, nil
	}

	type Timeformat struct {
		Layout     string
		Resolution time.Duration
	}

	var timeformats []Timeformat

	if len(s) < len("02 Jan 06 15:04 UTC") {
		return time.Time{}, 0, ErrInvalidTime
	}

	if '0' <= s[0] && s[0] <= '9' {
		switch s[2] {
		case '/':
			timeformats = []Timeformat{
				{time.Layout, time.Second},
			}
		case ' ':
			timeformats = []Timeformat{
				{time.RFC822, time.Minute},
				{time.RFC822Z, time.Minute},
			}
		}
	} else if s[0] == 'S' || s[0] == 'M' || s[0] == 'T' || s[0] == 'W' || s[0] == 'F' {
		switch s[3] {
		case ',':
			timeformats = []Timeformat{
				{time.RFC1123, time.Second},
				{time.RFC1123Z, time.Second},
			}
		case ' ':
			timeformats = []Timeformat{
				{time.ANSIC, time.Second},
				{time.UnixDate, time.Second},
				{time.RubyDate, time.Second},
			}
		default:
			timeformats = []Timeformat{
				{time.RFC850, time.Second},
			}
		}
	}

	fn := time.Parse
	if tz != nil {
		fn = func(layout, value string) (time.Time, error) {
			return time.ParseInLocation(layout, value, tz)
		}
	}

	for _, f := range timeformats {
		t, err := fn(f.Layout, s)
		if err == nil {
			return t, f.Resolution, nil
		}
	}
	return time.Time{}, 0, ErrInvalidTime
}

// ParseTime parses time string in Ayd way.
// This function supports RFC3339 and some variant formats.
func ParseTime(s string) (time.Time, error) {
	t, _, err := ParseTimeWithResolution(s, nil)
	return t, err
}
