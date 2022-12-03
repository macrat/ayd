package store

import (
	"strconv"
	"strings"
	"time"
)

type pathFragment interface {
	Build(t time.Time) string
	Len() int
	Match(s string, since, until time.Time) bool
}

type constFragment string

func (s constFragment) Build(_ time.Time) string {
	return string(s)
}

func (s constFragment) Len() int {
	return len(s)
}

func (s constFragment) Match(str string, _, _ time.Time) bool {
	return string(s) == str
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

func (y yearFragment) Match(s string, since, until time.Time) bool {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return false
	}
	if y.Short {
		n += 2000
	}
	return since.Year() <= n && n <= until.Year()
}

type monthFragment struct{}

func (m monthFragment) Build(t time.Time) string {
	return t.Format("01")
}

func (m monthFragment) Len() int {
	return 2
}

func (m monthFragment) Match(s string, since, until time.Time) bool {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || 12 < n {
		return false
	}

	if since.Year() == until.Year() {
		return int(since.Month()) <= n && n <= int(until.Month())
	}

	si := int(since.Month())
	ei := int(until.Month()) + 12

	return si <= n && n <= ei+12 || si <= n+12 && n+12 <= ei+12
}

type dayFragment struct{}

func (d dayFragment) Build(t time.Time) string {
	return t.Format("02")
}

func (d dayFragment) Len() int {
	return 2
}

func (d dayFragment) Match(s string, since, until time.Time) bool {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || 31 < n {
		return false
	}

	if since.Year() == until.Year() && since.Month() == until.Month() {
		return int(since.Day()) <= n && n <= int(until.Day())
	}

	if until.Sub(since) > (30+31)*24*time.Hour {
		// The period definitely contains all kind of numbers if it is more than 30+31 days.
		return true
	}

	since = since.Truncate(24 * time.Hour)
	until = until.Truncate(24 * time.Hour)

	// Try all days instead of calculation, because handling leap year is too complex.
	for t := since; t.Before(until); t = t.AddDate(0, 0, 1) {
		if t.Day() == n {
			return true
		}
	}
	return false
}

type hourFragment struct{}

func (h hourFragment) Build(t time.Time) string {
	return t.Format("15")
}

func (h hourFragment) Len() int {
	return 2
}

func (h hourFragment) Match(s string, since, until time.Time) bool {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || 23 < n {
		return false
	}

	if since.Truncate(24 * time.Hour).Equal(until.Truncate(24 * time.Hour)) {
		return int(since.Hour()) <= n && n <= int(until.Hour())
	}

	si := since.Hour()
	ei := until.Hour() + 24

	return si <= n && n <= ei || si <= n+24 && n+24 <= ei
}

type minuteFragment struct{}

func (m minuteFragment) Build(t time.Time) string {
	return t.Format("04")
}

func (m minuteFragment) Len() int {
	return 2
}

func (m minuteFragment) Match(s string, since, until time.Time) bool {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || 59 < n {
		return false
	}

	if since.Truncate(time.Hour).Equal(until.Truncate(time.Hour)) {
		return int(since.Minute()) <= n && n <= int(until.Minute())
	}

	si := since.Minute()
	ei := until.Minute() + 60

	return si <= n && n <= ei || si <= n+60 && n+60 <= ei
}

type Pattern struct {
	Unit time.Duration

	pattern   string
	fragments []pathFragment
}

func ParsePattern(s string) Pattern {
	p := Pattern{
		pattern: s,
	}

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

func (p Pattern) Build(t time.Time) string {
	ss := make([]string, len(p.fragments))
	for i, f := range p.fragments {
		ss[i] = f.Build(t)
	}
	return strings.Join(ss, "")
}

func (p Pattern) Len() int {
	n := 0
	for _, f := range p.fragments {
		n += f.Len()
	}
	return n
}

func (p Pattern) Match(s string, since, until time.Time) bool {
	l := 0
	for _, f := range p.fragments {
		r := l + f.Len()
		if len(s) < r {
			return false
		}
		if !f.Match(s[l:r], since, until) {
			return false
		}
		l = r
	}
	return true
}
