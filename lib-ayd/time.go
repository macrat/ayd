package ayd

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	timeformats []string

	ErrInvalidTime = errors.New("invalid format")
)

func init() {
	dfs := []string{
		"2006-01-02T",
		"2006-01-02_",
		"2006-01-02 ",
		"20060102 ",
		"20060102T",
		"20060102_",
	}
	tfs := []string{
		"15",
		"15:04",
		"15:04:05",
		"15:04:05.999999999",
		"1504",
		"150405",
		"150405.999999999",
	}
	zfs := []string{
		"Z07:00",
		"Z0700",
		"Z07",
	}
	for _, df := range dfs {
		for _, tf := range tfs {
			for _, zf := range zfs {
				timeformats = append(timeformats, df+tf+zf)
			}
		}
	}
	timeformats = append(
		timeformats,
		time.Layout,
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
	)
}

// ParseTime parses time string in Ayd way.
// This function supports RFC3339 and some variant formats.
func ParseTime(s string) (time.Time, error) {
	x := strings.ToUpper(strings.TrimSpace(s))
	for _, f := range timeformats {
		t, err := time.Parse(f, x)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: %q", ErrInvalidTime, s)
}
