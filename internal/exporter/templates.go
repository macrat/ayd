package exporter

import (
	"strings"
	"time"

	"github.com/macrat/ayd/internal/store/freeze"
)

var (
	templateFuncs = map[string]interface{}{
		"invert_incidents": func(xs []freeze.Incident) []freeze.Incident {
			rs := make([]freeze.Incident, len(xs))
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
		"format_latency": func(latency float64) string {
			return time.Duration(latency * float64(time.Millisecond)).String()
		},
	}
)
