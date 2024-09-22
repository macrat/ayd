package query

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

var (
	errValueNotOrderable = errors.New("value is not orderable")
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
	if t, err := api.ParseTime(s); err == nil {
		return timeValueMatcher{Op: opCode, Value: t, Str: s}, nil
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

type neverValueMatcher struct{}

func (neverValueMatcher) String() string {
	return "never"
}

func (neverValueMatcher) Match(value any) bool {
	return false
}

type stringValueMatcher struct {
	Op      operator
	Matcher stringMatcher
}

func (m stringValueMatcher) String() string {
	switch m.Op {
	case opNotEqual:
		return fmt.Sprintf("!=%q", m.Matcher)
	case opEqual:
		return fmt.Sprintf("=%q", m.Matcher)
	default:
		panic("unexpected operator")
	}
}

func newStringValueMatcher(ss []*string, opCode operator) valueMatcher {
	if opCode == opIncludes {
		return stringValueMatcher{
			Op:      opEqual,
			Matcher: newStringMatcher(append(append([]*string{nil}, ss...), nil)),
		}
	} else if opCode&(opEqual|opNotEqual) != 0 {
		m := newStringMatcher(ss)
		return stringValueMatcher{
			Op:      opCode,
			Matcher: m,
		}
	}

	return neverValueMatcher{}
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

	if m.Op == opNotEqual {
		return !m.Matcher.Match(s)
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
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32:
		n = float64(v.(int))
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
	case time.Time:
		n = float64(v.Unix())
	default:
		return false
	}

	if m.Op&opEqual != 0 && n == m.Value {
		return true
	}

	switch m.Op {
	case opLessThan:
		return n < m.Value
	case opGreaterThan:
		return n > m.Value
	case opNotEqual:
		return n != m.Value
	case opIncludes:
		return n == m.Value
	}

	return false
}

type timeValueMatcher struct {
	Op    operator
	Value time.Time
	Str   string
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
	case opIncludes:
		return t.Equal(m.Value)
	}

	return false
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
	case opIncludes:
		return d == m.Value
	}

	return false
}
