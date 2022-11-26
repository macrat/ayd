package ayd

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	dateformats = []string{
		"2006-01-02T",
		"2006-01-02_",
		"2006-01-02 ",
		"20060102 ",
		"20060102T",
		"20060102_",
	}
	timeformats = []string{
		"15:04:05",
		"15:04:05.999999999",
		"150405",
		"150405.999999999",
	}
	zoneformats = []string{
		"Z07:00",
		"Z0700",
		"Z07",
	}

	ErrInvalidTime = errors.New("invalid format")
)

// ParseTime parses time string in Ayd way.
// This function supports RFC3339 and some variant formats.
func ParseTime(s string) (time.Time, error) {
	x := strings.ToUpper(strings.TrimSpace(s))
	for _, df := range dateformats {
		for _, tf := range timeformats {
			for _, zf := range zoneformats {
				t, err := time.Parse(df+tf+zf, x)
				if err == nil {
					return t, nil
				}
			}
		}
	}
	return time.Time{}, fmt.Errorf("%w: %q", ErrInvalidTime, s)
}
