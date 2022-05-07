package endpoint

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/url"
	"sort"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

//go:embed templates/base.html
var baseHTMLTemplateStr string

var baseHTMLTemplate = template.Must(template.New("base.html").Funcs(templateFuncs).Parse(baseHTMLTemplateStr))

func loadHTMLTemplate(s string) *template.Template {
	return template.Must(
		template.Must(baseHTMLTemplate.Clone()).Parse(s),
	)
}

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
		"pad_records": func(length int, rs []api.Record) []struct{} {
			if len(rs) >= length {
				return []struct{}{}
			}
			return make([]struct{}, length-len(rs))
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
		"is_debased": func(s api.Status) bool {
			return s == api.StatusDebased
		},
		"is_healthy": func(s api.Status) bool {
			return s == api.StatusHealthy
		},
		"to_lower": func(s fmt.Stringer) string {
			return strings.ToLower(s.String())
		},
		"to_camel": func(s fmt.Stringer) string {
			lower := strings.ToLower(s.String())
			return strings.ToUpper(lower[:1]) + lower[1:]
		},
		"time2str": func(t time.Time) string {
			return t.Format(time.RFC3339)
		},
		"time2rfc822": func(t time.Time) string {
			return t.Format(time.RFC822)
		},
		"url_unescape": func(u *url.URL) string {
			s, err := url.PathUnescape(u.String())
			if err != nil {
				return u.String()
			}
			return s
		},
		"latency_graph": func(rs []api.Record) string {
			if len(rs) == 0 {
				return ""
			}

			maxLatency := 0.0
			for _, r := range rs {
				l := r.Latency.Seconds()
				if l > maxLatency {
					maxLatency = l
				}
			}

			offset := 20 - len(rs)

			result := make([]string, len(rs)+2)
			result[0] = fmt.Sprintf("M%d,1 %d,%v", offset, offset, 1-rs[0].Latency.Seconds()/maxLatency)

			for i, r := range rs {
				result[i+1] = fmt.Sprintf("%d.5,%v", offset+i, 1-r.Latency.Seconds()/maxLatency)
			}

			result[len(result)-1] = "h0.5V1"

			return strings.Join(result, " ")
		},
		"calculate_summary": func(hs map[string]api.ProbeHistory) []statusSummary {
			counts := make(map[api.Status]int)
			total := 0
			for _, h := range hs {
				if _, ok := counts[h.Status]; ok {
					counts[h.Status]++
				} else {
					counts[h.Status] = 1
				}
				total++
			}
			result := make([]statusSummary, 0, len(counts))
			for s, c := range counts {
				result = append(result, statusSummary{s, float32(c) * 100 / float32(total), 0})
			}
			sort.Slice(result, func(i, j int) bool {
				x, y := result[i], result[j]
				switch {
				case x.Percent != y.Percent:
					return x.Percent > y.Percent
				case x.Status == y.Status:
					return false
				case x.Status == api.StatusUnknown:
					return false
				case y.Status == api.StatusUnknown:
					return true
				default:
					return x.Status < y.Status
				}
			})
			var sum float32 = 0.0
			for i := range result {
				result[i].Cumulative += sum
				sum += result[i].Percent
			}
			return result
		},
	}
)

type statusSummary struct {
	Status     api.Status
	Percent    float32
	Cumulative float32
}
