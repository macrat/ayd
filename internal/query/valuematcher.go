package query

import (
	"fmt"
	"strconv"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type valueMatcher interface {
	Match(value any) bool
}

type stringValueMatcher struct {
	Not     bool
	Matcher stringMatcher
}

func (m stringValueMatcher) Match(value any) bool {
	var s string
	if v, ok := value.(string); ok {
		s = v
	} else {
		s = fmt.Sprintf("%v", value)
	}

	if m.Not {
		return !m.Matcher.Match(s)
	}
	return m.Matcher.Match(s)
}

type numberValueMatcher struct {
	Op    operator
	Value float64
}

func (m numberValueMatcher) Match(value any) bool {
	var n float64
	switch v := value.(type) {
	case float64:
		n = v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32:
		n = float64(v.(int))
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			n = f
		} else {
			return false
		}
	case time.Duration:
		n = float64(v.Microseconds()) / 1000
	case time.Time:
		n = float64(v.Unix())
	default:
		return false
	}

	switch m.Op {
	case opEqual:
		return n == m.Value
	case opLessThan:
		return n < m.Value
	case opGreaterThan:
		return n > m.Value
	case opLessEqual:
		return n <= m.Value
	case opGreaterEqual:
		return n >= m.Value
	case opNotEqual:
		return n != m.Value
	}

	return false
}

type timeValueMatcher struct {
	Op    operator
	Value time.Time
}

func (m timeValueMatcher) Match(value any) bool {
	var t time.Time
	switch v := value.(type) {
	case time.Time:
		t = v
	case string:
		if ts, err := api.ParseTime(v); err == nil {
			t = ts
		} else {
			return false
		}
	case int:
		t = time.Unix(int64(v), 0)
	case float64:
		t = time.Unix(int64(v), 0)
	default:
		return false
	}

	if m.Op&opEqual != 0 && t.Equal(m.Value) {
		return true
	}

	switch m.Op {
	case opLessThan:
		return t.Before(m.Value)
	case opGreaterThan:
		return t.After(m.Value)
	case opNotEqual:
		return !t.Equal(m.Value)
	}

	return false
}

type durationValueMatcher struct {
	Op    operator
	Value time.Duration
}

func (m durationValueMatcher) Match(value any) bool {
	var d time.Duration
	switch v := value.(type) {
	case time.Duration:
		d = v
	case string:
		if ds, err := time.ParseDuration(v); err == nil {
			d = ds
		} else {
			return false
		}
	case int:
		d = time.Duration(v) * time.Millisecond
	case float64:
		d = time.Duration(v) * time.Millisecond
	default:
		return false
	}

	if m.Op&opEqual != 0 && d == m.Value {
		return true
	}

	switch m.Op {
	case opLessThan:
		return d < m.Value
	case opGreaterThan:
		return d > m.Value
	case opNotEqual:
		return d != m.Value
	}

	return false
}
