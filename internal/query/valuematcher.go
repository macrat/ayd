package query

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	errValueNotOrderable  = errors.New("value is not orderable")
	errIncorrectValueType = errors.New("incorrect value type")
)

type valueMatcher interface {
	fmt.Stringer

	Match(value any) bool
}

func newOrderingValueMatcher(ss []*string, opCode operator) (valueMatcher, error) {
	if len(ss) != 1 || ss[0] == nil {
		return nil, errValueNotOrderable
	}
	s := *ss[0]

	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return numberValueMatcher{Op: opCode, Value: n, Str: s}, nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		return durationValueMatcher{Op: opCode, Value: d, Str: s}, nil
	}
	if t, err := parseTimeValueMatcher(opCode, s); err == nil {
		return t, nil
	}
	return nil, errValueNotOrderable
}

func newValueMatcher(ss []*string, opCode operator) valueMatcher {
	if len(ss) == 1 && ss[0] != nil {
		if m, err := newOrderingValueMatcher(ss, opCode); err == nil {
			return m
		}
	}

	return newStringValueMatcher(ss, opCode)
}

type anyValueMatcher struct{}

func (anyValueMatcher) String() string {
	return "ANY"
}

func (anyValueMatcher) Match(value any) bool {
	return true
}

type stringValueMatcher struct {
	Op      operator
	Matcher stringMatcher
}

func (m stringValueMatcher) String() string {
	return fmt.Sprintf("=%q", m.Matcher)
}

func newStringValueMatcher(ss []*string, opCode operator) valueMatcher {
	if opCode == opIncludes {
		return stringValueMatcher{
			Op:      opEqual,
			Matcher: newStringMatcher(append(append([]*string{nil}, ss...), nil)),
		}
	} else {
		return stringValueMatcher{
			Op:      opCode,
			Matcher: newStringMatcher(ss),
		}
	}
}

func (m stringValueMatcher) Match(value any) bool {
	var s string
	if v, ok := value.(string); ok {
		s = v
	} else if stringer, ok := value.(fmt.Stringer); ok {
		s = stringer.String()
	} else {
		s = fmt.Sprintf("%v", value)
	}

	return m.Matcher.Match(s)
}

type numberValueMatcher struct {
	Op    operator
	Value float64
	Str   string
}

func (m numberValueMatcher) String() string {
	return fmt.Sprintf("%s%f", m.Op, m.Value)
}

func (m numberValueMatcher) Match(value any) bool {
	var n float64
	switch v := value.(type) {
	case float64:
		n = v
	case int:
		n = float64(v)
	case int8:
		n = float64(v)
	case int16:
		n = float64(v)
	case int32:
		n = float64(v)
	case int64:
		n = float64(v)
	case uint:
		n = float64(v)
	case uint8:
		n = float64(v)
	case uint16:
		n = float64(v)
	case uint32:
		n = float64(v)
	case uint64:
		n = float64(v)
	case float32:
		n = float64(v)
	case string:
		if m.Op == opIncludes && strings.Contains(v, m.Str) {
			return true
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			n = f
		} else {
			return false
		}
	case time.Duration:
		n = float64(v.Microseconds()) / 1000
	default:
		return false
	}

	if m.Op&opEqual != 0 && n == m.Value {
		return true
	}

	if m.Op&opLessThan != 0 {
		return n < m.Value
	}
	if m.Op&opGreaterThan != 0 {
		return n > m.Value
	}
	return n == m.Value
}

type timeValueMatcher struct {
	Op         operator
	Value      time.Time
	Resolution time.Duration
	Str        string
}

type timeformat struct {
	Layout     string
	Resolution time.Duration
}

var timeformats = []timeformat{}

func init() {
	dfs := []string{
		"2006-01-02T",
		"2006-01-02_",
		"2006-01-02 ",
		"20060102 ",
		"20060102T",
		"20060102_",
	}
	tfs := []timeformat{
		{"15", time.Hour},
		{"15:04", time.Minute},
		{"15:04:05", time.Second},
		{"15:04:05.999999999", time.Nanosecond},
		{"1504", time.Minute},
		{"150405", time.Second},
		{"150405.999999999", time.Nanosecond},
	}
	zfs := []string{
		"Z07:00",
		"Z0700",
		"Z07",
	}
	for _, df := range dfs {
		for _, tf := range tfs {
			for _, zf := range zfs {
				timeformats = append(timeformats, timeformat{
					Layout:     df + tf.Layout + zf,
					Resolution: tf.Resolution,
				}, timeformat{
					Layout:     df + tf.Layout,
					Resolution: tf.Resolution,
				})
			}
		}
	}
	timeformats = append(
		timeformats,
		timeformat{"2006-01-02", 24 * time.Hour},
		timeformat{"2006/01/02", 24 * time.Hour},
		timeformat{"2006/1/2", 24 * time.Hour},
		timeformat{time.Layout, time.Second},
		timeformat{time.ANSIC, time.Second},
		timeformat{time.UnixDate, time.Second},
		timeformat{time.RubyDate, time.Second},
		timeformat{time.RFC822, time.Second},
		timeformat{time.RFC822Z, time.Second},
		timeformat{time.RFC850, time.Second},
		timeformat{time.RFC1123, time.Second},
		timeformat{time.RFC1123Z, time.Second},
	)
}

func parseTimeValueMatcher(op operator, value string) (timeValueMatcher, error) {
	x := strings.ToUpper(strings.TrimSpace(value))
	for _, f := range timeformats {
		t, err := time.Parse(f.Layout, x)
		if err == nil {
			if op == opGreaterThan || op == opLessEqual {
				t = t.Add(f.Resolution).Add(-1)
			}
			return timeValueMatcher{Op: op, Value: t, Resolution: f.Resolution, Str: value}, nil
		}
	}

	extraformats := []timeformat{
		{"15:04:05.999999999", time.Nanosecond},
		{"15:04:05", time.Second},
		{"15:04", time.Minute},
	}
	for _, f := range extraformats {
		t, err := time.Parse(f.Layout, x)
		if err == nil {
			if op == opGreaterThan || op == opLessEqual {
				t = t.Add(f.Resolution).Add(-1)
			}
			year, month, day := time.Now().Date()
			t = t.AddDate(year, int(month)-1, day-1)
			return timeValueMatcher{Op: op, Value: t, Resolution: f.Resolution, Str: value}, nil
		}
	}

	return timeValueMatcher{}, errIncorrectValueType
}

func (m timeValueMatcher) String() string {
	return fmt.Sprintf("%s%s", m.Op, m.Value.Format(time.RFC3339))
}

func (m timeValueMatcher) Match(value any) bool {
	var t time.Time
	switch v := value.(type) {
	case time.Time:
		t = v
	case string:
		if m.Op == opIncludes && strings.Contains(v, m.Str) {
			return true
		}
		if ts, err := parseTimeValueMatcher(opIncludes, v); err == nil {
			t = ts.Value
		} else {
			return false
		}
	case int:
		t = time.Unix(int64(v), 0)
	case float64:
		t = time.UnixMicro(int64(v * 1e6))
	default:
		return false
	}

	if m.Op&opEqual != 0 && t.Equal(m.Value) {
		return true
	}

	if m.Op&opLessThan != 0 {
		return t.Before(m.Value)
	}
	if m.Op&opGreaterThan != 0 {
		return t.After(m.Value)
	}
	return !t.Before(m.Value) && t.Before(m.Value.Add(m.Resolution))
}

type durationValueMatcher struct {
	Op    operator
	Value time.Duration
	Str   string
}

func (m durationValueMatcher) String() string {
	return fmt.Sprintf("%s%s", m.Op, m.Value)
}

func (m durationValueMatcher) Match(value any) bool {
	var d time.Duration
	switch v := value.(type) {
	case time.Duration:
		d = v
	case string:
		if m.Op == opIncludes && strings.Contains(v, m.Str) {
			return true
		}
		if ds, err := time.ParseDuration(v); err == nil {
			d = ds
		} else {
			return false
		}
	case int:
		d = time.Duration(v) * time.Millisecond
	case float64:
		d = time.Duration(v * float64(time.Millisecond))
	default:
		return false
	}

	if m.Op&opEqual != 0 && d == m.Value {
		return true
	}

	if m.Op&opLessThan != 0 {
		return d < m.Value
	}
	if m.Op&opGreaterThan != 0 {
		return d > m.Value
	}
	return d == m.Value
}
