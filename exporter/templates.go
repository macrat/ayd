package exporter

import (
	"strings"
)

var (
	templateFuncs = map[string]interface{}{
		"each_runes": func(s string) []string {
			r := make([]string, len(s))
			for i, c := range []rune(s) {
				r[i] = string(c)
			}
			return r
		},
		"invert_incidents": func(xs []frozenIncident) []frozenIncident {
			rs := make([]frozenIncident, len(xs))
			for i, x := range xs {
				rs[len(xs)-i-1] = x
			}
			return rs
		},
		"break_text": func(s string, width int) []string {
			r := []string{}
			for start := 0; start < len(s); start += width {
				end := start + width
				if end >= len(s) {
					end = len(s)
				}
				r = append(r, s[start:end])
			}
			return r
		},
		"align_center": func(s string, width int) string {
			if len(s) > width {
				return s
			}
			return strings.Repeat(" ", (width-len(s))/2) + s
		},
	}
)
