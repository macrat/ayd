package store

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type pathFragment interface {
	Build(t time.Time) string
	Len() int
	Glob() string
	FillTimePattern(s string, tp *timePattern) (ok bool)
}

type constFragment string

func (c constFragment) Build(_ time.Time) string {
	return string(c)
}

func (c constFragment) Len() int {
	return len(c)
}

func (c constFragment) Glob() string {
	return string(c)
}

func (c constFragment) FillTimePattern(s string, tp *timePattern) (ok bool) {
	return s == string(c)
}

type yearFragment struct {
	Short bool
}

func (y yearFragment) Build(t time.Time) string {
	if y.Short {
		return t.Format("06")
	} else {
		return strconv.Itoa(t.Year())
	}
}

func (y yearFragment) Len() int {
	if y.Short {
		return 2
	} else {
		return 4
	}
}

func (y yearFragment) Glob() string {
	if y.Short {
		return "[0-9][0-9]"
	} else {
		return "[0-9][0-9][0-9][0-9]"
	}
}

func (y yearFragment) FillTimePattern(s string, tp *timePattern) (ok bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return false
	}

	if y.Short {
		n += 2000
	}

	if tp.Year >= 0 && tp.Year != n {
		return false
	}

	tp.Year = n
	return true
}

type monthFragment struct{}

func (m monthFragment) Build(t time.Time) string {
	return t.Format("01")
}

func (m monthFragment) Len() int {
	return 2
}

func (m monthFragment) Glob() string {
	return "[0-1][0-9]"
}

func (m monthFragment) FillTimePattern(s string, tp *timePattern) (ok bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || 12 < n {
		return false
	}

	if tp.Month >= 1 && tp.Month != n {
		return false
	}

	tp.Month = n
	return true
}

type dayFragment struct{}

func (d dayFragment) Build(t time.Time) string {
	return t.Format("02")
}

func (d dayFragment) Len() int {
	return 2
}

func (d dayFragment) Glob() string {
	return "[0-3][0-9]"
}

func (d dayFragment) FillTimePattern(s string, tp *timePattern) (ok bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || 31 < n {
		return false
	}

	if tp.Day >= 1 && tp.Day != n {
		return false
	}

	tp.Day = n
	return true
}

type hourFragment struct{}

func (h hourFragment) Build(t time.Time) string {
	return t.Format("15")
}

func (h hourFragment) Len() int {
	return 2
}

func (h hourFragment) Glob() string {
	return "[0-2][0-9]"
}

func (h hourFragment) FillTimePattern(s string, tp *timePattern) (ok bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || 23 < n {
		return false
	}

	if tp.Hour >= 0 && tp.Hour != n {
		return false
	}

	tp.Hour = n
	return true
}

type minuteFragment struct{}

func (m minuteFragment) Build(t time.Time) string {
	return t.Format("04")
}

func (m minuteFragment) Len() int {
	return 2
}

func (m minuteFragment) Glob() string {
	return "[0-5][0-9]"
}

func (m minuteFragment) FillTimePattern(s string, tp *timePattern) (ok bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || 59 < n {
		return false
	}

	if tp.Minute >= 0 && tp.Minute != n {
		return false
	}

	tp.Minute = n
	return true
}

type PathPattern struct {
	pattern   string
	fragments []pathFragment
}

func ParsePathPattern(s string) PathPattern {
	p := PathPattern{pattern: s}

	var buf []string
	left := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			i++

			if len(s) <= i {
				break
			}

			if s[i] == '%' {
				buf = append(buf, s[left:i])
				left = i + 1
				continue
			}

			buf = append(buf, s[left:i-1])
			left = i + 1
			if c := constFragment(strings.Join(buf, "")); c != "" {
				p.fragments = append(p.fragments, c)
			}
			buf = nil

			switch s[i] {
			case 'Y':
				p.fragments = append(p.fragments, yearFragment{false})
			case 'y':
				p.fragments = append(p.fragments, yearFragment{true})
			case 'm':
				p.fragments = append(p.fragments, monthFragment{})
			case 'd':
				p.fragments = append(p.fragments, dayFragment{})
			case 'H':
				p.fragments = append(p.fragments, hourFragment{})
			case 'M':
				p.fragments = append(p.fragments, minuteFragment{})
			default:
				buf = append(buf, "%", string(s[i]))
			}
		}
	}
	if c := constFragment(strings.Join(append(buf, s[left:]), "")); c != "" {
		p.fragments = append(p.fragments, c)
	}

	return p
}

func (p PathPattern) String() string {
	return p.pattern
}

func (p PathPattern) IsEmpty() bool {
	return len(p.fragments) == 0
}

func (p PathPattern) Build(t time.Time) string {
	ss := make([]string, len(p.fragments))
	for i, f := range p.fragments {
		ss[i] = f.Build(t)
	}
	return strings.Join(ss, "")
}

func (p PathPattern) parseTimePattern(filename string) (tp timePattern, ok bool) {
	tp = emptyTimePattern

	l := 0
	for _, f := range p.fragments {
		r := l + f.Len()
		if r > len(filename) {
			return tp, false
		}

		if !f.FillTimePattern(filename[l:r], &tp) {
			return tp, false
		}

		l = r
	}

	return tp, len(filename) == l
}

func (p PathPattern) Match(filename string, since, until time.Time) bool {
	if p.IsEmpty() {
		return filename == ""
	}

	tp, ok := p.parseTimePattern(filename)
	if !ok {
		return false
	}

	max := tp.Exec(until, maxTimePattern)
	min := tp.Exec(since, minTimePattern)

	return !since.After(max) && !min.After(until)
}

func (p PathPattern) Glob() string {
	ss := make([]string, len(p.fragments))
	for i, f := range p.fragments {
		ss[i] = f.Glob()
	}
	return strings.Join(ss, "")
}

// ListAll returns all log file pathes.
// The result is sorted by time order.
func (p PathPattern) ListAll() []string {
	xs, err := filepath.Glob(p.Glob())
	if err != nil {
		return nil
	}

	rs := make([]string, 0, len(xs))
	tps := make([]timePattern, 0, len(xs))

	for _, x := range xs {
		if tp, ok := p.parseTimePattern(x); ok {
			rs = append(rs, x)
			tps = append(tps, tp)
		}
	}

	sort.Slice(rs, func(i, j int) bool {
		return tps[i].Less(tps[j])
	})

	return rs
}

// ListBetween returns log file pathes.
// The result is filtered by since and until query, but not sorted.
func (p PathPattern) ListBetween(since, until time.Time) []string {
	xs, err := filepath.Glob(p.Glob())
	if err != nil {
		return nil
	}

	rs := make([]string, 0, len(xs))

	for _, x := range xs {
		if p.Match(x, since, until) {
			rs = append(rs, x)
		}
	}

	return rs
}

type timePattern struct {
	Year   int
	Month  int
	Day    int
	Hour   int
	Minute int
}

var (
	emptyTimePattern = timePattern{-1, -1, -1, -1, -1}
	minTimePattern   = timePattern{0, 1, 1, 0, 0}
	maxTimePattern   = timePattern{9999, 12, 31, 23, 59}
)

func (p timePattern) Exec(t time.Time, base timePattern) time.Time {
	useCopy := false
	r := base

	if p.Minute >= 0 {
		r.Minute = p.Minute
		useCopy = true
	}

	if p.Hour >= 0 {
		r.Hour = p.Hour
		useCopy = true
	} else if useCopy {
		r.Hour = t.Hour()
	}

	if p.Day >= 1 {
		r.Day = p.Day
		useCopy = true
	} else if useCopy {
		r.Day = t.Day()
	}

	if p.Month >= 1 {
		r.Month = p.Month
		useCopy = true
	} else if useCopy {
		r.Month = int(t.Month())
	}

	if p.Year >= 0 {
		r.Year = p.Year
		//useCopy = true // No need this.
	} else if useCopy {
		r.Year = t.Year()
	}

	return r.Time(t.Location())
}

func (p timePattern) Time(loc *time.Location) time.Time {
	return time.Date(
		p.Year,
		time.Month(p.Month),
		p.Day,
		p.Hour,
		p.Minute,
		0,
		0,
		loc,
	)
}

func (p timePattern) Less(x timePattern) bool {
	return p.Year < x.Year || p.Month < x.Month || p.Day < x.Day || p.Hour < x.Hour || p.Minute < x.Minute
}
