package endpoint

import (
	_ "embed"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
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
		"time_range": func(rs []api.Record) timeRange {
			switch len(rs) {
			case 0:
				return timeRange{}
			case 1:
				return timeRange{
					Newest: rs[0].Time,
				}
			default:
				return timeRange{
					Oldest: rs[0].Time,
					Newest: rs[len(rs)-1].Time,
				}
			}
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
		"is_degrade": func(s api.Status) bool {
			return s == api.StatusDegrade
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
		"time2humanize": func(t time.Time) string {
			return humanize.Time(t)
		},
		"latency2float": func(d time.Duration) float64 {
			return float64(d.Microseconds()) / 1000.0
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
			builder := newStatusSummaryBuilder()
			for _, h := range hs {
				builder.Add(h.Status)
			}
			return builder.Build()
		},
		"target_summary": func(rs []api.Record) []statusSummary {
			builder := newStatusSummaryBuilder()
			for _, r := range rs {
				builder.Add(r.Status)
			}
			return builder.Build()
		},
	}
)

type statusSummary struct {
	Status     api.Status
	Percent    float32
	Cumulative float32
	IsLast     bool
}

type statusSummaryBuilder struct {
	Count map[api.Status]int
	Total int
}

func newStatusSummaryBuilder() *statusSummaryBuilder {
	return &statusSummaryBuilder{
		Count: make(map[api.Status]int),
	}
}

func (b *statusSummaryBuilder) Add(s api.Status) {
	if _, ok := b.Count[s]; ok {
		b.Count[s]++
	} else {
		b.Count[s] = 1
	}
	b.Total++
}

func (b *statusSummaryBuilder) Build() []statusSummary {
	// TODO: add test for this function

	result := make([]statusSummary, len(b.Count))
	i := 0
	for s, c := range b.Count {
		result[i] = statusSummary{s, float32(c) * 100 / float32(b.Total), 0, false}
		i++
	}
	sort.Slice(result, func(i, j int) bool {
		x, y := result[i], result[j]
		if x.Percent == y.Percent {
			return x.Status < y.Status
		} else {
			return x.Percent > y.Percent
		}
	})
	var sum float32 = 0.0
	for i := range result {
		result[i].Cumulative += sum
		sum += result[i].Percent
	}
	if len(result) > 0 {
		result[len(result)-1].IsLast = true
	}
	return result
}

type timeRange struct {
	Oldest time.Time
	Newest time.Time
}
