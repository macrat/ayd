package endpoint

import (
	"fmt"
	"net/url"
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
		"to_lower": func(s fmt.Stringer) string {
			return strings.ToLower(s.String())
		},
		"time2str": func(t time.Time) string {
			return t.Format(time.RFC3339)
		},
		"url_unescape": func(u *url.URL) string {
			s, err := url.PathUnescape(u.String())
			if err != nil {
				return u.String()
			}
			return s
		},
		"latency_graph": func(rs []api.Record) string {
			maxLatency := 0.0
			for _, r := range rs {
				l := r.Latency.Seconds()
				if l > maxLatency {
					maxLatency = l
				}
			}

			offset := store.PROBE_HISTORY_LEN - len(rs)

			result := make([]string, len(rs)+2)
			result[0] = fmt.Sprintf("M%d,1 %d,%v", offset, offset, 1-rs[0].Latency.Seconds()/maxLatency)

			for i, r := range rs {
				result[i+1] = fmt.Sprintf("%d.5,%v", offset+i, 1-r.Latency.Seconds()/maxLatency)
			}

			result[len(result)-1] = "h0.5V1"

			return strings.Join(result, " ")
		},
	}
)
