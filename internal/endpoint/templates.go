package endpoint

import (
	_ "embed"
	"fmt"
	"html/template"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	api "github.com/macrat/ayd/lib-ayd"
	"golang.org/x/text/width"
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
			ss := []string{}
			var buf []rune
			for i, r := range []rune(s) {
				if i > 0 && len(buf) >= width {
					ss = append(ss, string(buf))
					buf = buf[:0]
				}
				buf = append(buf, r)
			}
			if len(buf) > 0 {
				ss = append(ss, string(buf))
			}
			return ss
		},
		"align_center": func(s string, width_ int) string {
			length := 0
			for _, r := range []rune(s) {
				switch width.LookupRune(r).Kind() {
				case width.EastAsianWide, width.EastAsianFullwidth:
					length += 2
				default:
					length += 1
				}
			}

			if length > width_ {
				return s
			}
			return strings.Repeat(" ", (width_-length)/2) + s
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
		"time2str_date": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		"time2str_time": func(t time.Time) string {
			return t.Format("15:04:05")
		},
		"time2str_zone": func(t time.Time) string {
			return t.Format("Z07:00")
		},
		"time2rfc822": func(t time.Time) string {
			return t.Format(time.RFC822)
		},
		"time2humanize": func(t time.Time) string {
			return humanize.Time(t)
		},
		"latency2str": func(d time.Duration) string {
			format := func(n int64) string {
				switch {
				case n < 10*10000:
					return fmt.Sprintf("%.3f", float64(n)/10000)
				case n < 100*10000:
					return fmt.Sprintf("%.2f", float64(n)/10000)
				default:
					return fmt.Sprintf("%.1f", float64(n)/10000)
				}
			}

			n := d.Nanoseconds()
			switch {
			case n < 1000:
				return "0"
			case n < 1000*1000:
				return fmt.Sprintf("%.3f", float64(n)/1000/1000) + "ms"
			case n < 1000*1000*1000:
				return format(n/100) + "ms"
			case n < 60*1000*1000*1000:
				return format(n/1000/100) + "s"
			default:
				m := d.Truncate(time.Minute).String()
				s := float64(d.Milliseconds()%60000) / 1000
				return m[:len(m)-2] + fmt.Sprintf("%.0fs", s)
			}
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
		"url2uuid": func(u *api.URL) string {
			return uuid.NewSHA1(uuid.NameSpaceURL, []byte(u.String())).String()
		},
		"uint2humanize": func(n uint64) string {
			var ss []string
			for n > 999 {
				s := strconv.FormatUint(n%1000, 10)
				switch len(s) {
				case 2:
					s = "0" + s
				case 1:
					s = "00" + s
				}
				ss = append(ss, s)
				n /= 1000
			}
			ss = append(ss, strconv.FormatUint(n%1000, 10))
			for i := 0; i < len(ss)/2; i++ {
				j := len(ss) - i - 1
				ss[i], ss[j] = ss[j], ss[i]
			}
			return strings.Join(ss, ",")
		},
		"extra2jsons": func(extra map[string]any) (xs []extraPair) {
			if len(extra) == 0 {
				return nil
			}
			xs = make([]extraPair, 0, len(extra))
			for k, v := range extra {
				b, err := json.Marshal(v)
				if err == nil {
					xs = append(xs, extraPair{
						Key:   k,
						Value: string(b),
					})
				}
			}
			sort.Slice(xs, func(i, j int) bool {
				return xs[i].Key < xs[j].Key
			})
			xs[len(xs)-1].IsLast = true
			return xs
		},
	}
)

type statusSummary struct {
	Status     api.Status
	Count      int
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
		result[i] = statusSummary{s, c, float32(c) * 100 / float32(b.Total), 0, false}
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

type extraPair struct {
	Key    string
	Value  string
	IsLast bool
}
