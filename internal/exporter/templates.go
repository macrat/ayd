package exporter

import (
	"strings"
	"time"

	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
)

var (
	templateFuncs = map[string]interface{}{
		"sort_history": func(hm map[string]api.ProbeHistory) []api.ProbeHistory {
			hs := make([]api.ProbeHistory, 0, len(hm))
			for _, h := range hm {
				hs = append(hs, h)
			}
			api.SortProbeHistories(hs)
			return hs
		},
		"invert_incidents": func(xs []api.Incident) []api.Incident {
			rs := make([]api.Incident, len(xs))
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
		"pad_records": func(rs []api.Record) []struct{} {
			if len(rs) >= store.PROBE_HISTORY_LEN {
				return []struct{}{}
			}
			return make([]struct{}, store.PROBE_HISTORY_LEN-len(rs))
		},
		"is_unknown": func(s api.Status) bool {
			return s == api.StatusUnknown
		},
		"is_aborted": func(s api.Status) bool {
			return s == api.StatusAborted
		},
		"is_failure": func(s api.Status) bool {
			return s == api.StatusFailure
		},
		"is_healthy": func(s api.Status) bool {
			return s == api.StatusHealthy
		},
		"time2str": func(t time.Time) string {
			return t.Format(time.RFC3339)
		},
	}
)
