package query

import (
	"fmt"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type Query interface {
	fmt.Stringer

	Match(api.Record) bool
	Optimize() Query
	TimeRange() (*time.Time, *time.Time)
}

type queryInverter interface {
	invert() Query
}

type TimeRange struct {
	Start *time.Time
	End   *time.Time
}

func ParseQuery(query string) Query {
	stack := []*And{{paren: true}}
	tok := newTokenizer(query)

	orMode := false
	notMode := false

	pushQuery := func(q Query) {
		if notMode {
			q = &Not{Query: q}
		}
		if orMode {
			stack[len(stack)-1].Queries = []Query{
				&Or{Queries: []Query{
					&And{Queries: stack[len(stack)-1].Queries},
					q,
				}},
			}
		} else {
			if len(stack[len(stack)-1].Queries) >= 1 {
				if or, ok := stack[len(stack)-1].Queries[len(stack[len(stack)-1].Queries)-1].(*Or); ok {
					last := or.Queries[len(or.Queries)-1]
					if and, ok := last.(*And); ok {
						and.Queries = append(and.Queries, q)
					} else {
						or.Queries[len(or.Queries)-1] = &And{Queries: []Query{last, q}}
					}
					return
				}
			}
			stack[len(stack)-1].Queries = append(stack[len(stack)-1].Queries, q)
		}

		orMode = false
		notMode = false
	}

	for tok.Scan() {
		switch tok.Token().Type {
		case lparenToken:
			and := &And{paren: true}
			pushQuery(and)
			stack = append(stack, and)
		case rparenToken:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			} else {
				stack = []*And{{
					Queries: []Query{stack[0]},
					paren:   true,
				}}
			}
			orMode = false
			notMode = false
		case orToken:
			orMode = true
			notMode = false
		case notToken:
			notMode = !notMode
		case simpleKeywordToken:
			if tok.Token().Not {
				pushQuery(&Not{&SimpleQuery{Value: tok.Token().Value}})
			} else {
				pushQuery(&SimpleQuery{Value: tok.Token().Value})
			}
		case fieldKeywordToken:
			if tok.Token().Not {
				pushQuery(&Not{&FieldQuery{Key: tok.Token().Key, Value: tok.Token().Value}})
			} else {
				pushQuery(&FieldQuery{Key: tok.Token().Key, Value: tok.Token().Value})
			}
		}
	}

	return stack[0].Optimize()
}

type And struct {
	Queries []Query
	paren   bool
}

func (q *And) String() string {
	if len(q.Queries) == 0 {
		return "(AND)"
	}

	qs := make([]string, len(q.Queries))
	for i, query := range q.Queries {
		qs[i] = query.String()
	}
	return "(AND " + strings.Join(qs, " ") + ")"
}

func (q *And) Match(r api.Record) bool {
	for _, query := range q.Queries {
		if !query.Match(r) {
			return false
		}
	}
	return true
}

func (q *And) Optimize() Query {
	qs := make([]Query, 0, len(q.Queries))
	for _, q := range q.Queries {
		q = q.Optimize()
		if and, ok := q.(*And); ok {
			qs = append(qs, and.Queries...)
		} else {
			qs = append(qs, q)
		}
	}
	if len(qs) == 1 {
		return qs[0]
	}
	return &And{Queries: qs, paren: q.paren}
}

func (q *And) invert() Query {
	qs := make([]Query, len(q.Queries))
	for i, q := range q.Queries {
		qs[i] = &Not{Query: q}
	}
	return (&Or{Queries: qs}).Optimize()
}

func (q *And) TimeRange() (*time.Time, *time.Time) {
	var start, end *time.Time

	// Get minimum range.
	// For example: "time>2025-01-01 AND time>2025-06-01" is the same as "time>2025-06-01"
	for _, query := range q.Queries {
		s, e := query.TimeRange()
		if s != nil && (start == nil || s.After(*start)) {
			start = s
		}
		if e != nil && (end == nil || e.Before(*end)) {
			end = e
		}
	}

	return start, end
}

type Or struct {
	Queries []Query
}

func (q *Or) String() string {
	if len(q.Queries) == 0 {
		return "(OR)"
	}

	qs := make([]string, len(q.Queries))
	for i, query := range q.Queries {
		qs[i] = query.String()
	}
	return "(OR " + strings.Join(qs, " ") + ")"
}

func (q *Or) Match(r api.Record) bool {
	for _, query := range q.Queries {
		if query.Match(r) {
			return true
		}
	}
	return false
}

func (q *Or) Optimize() Query {
	qs := make([]Query, 0, len(q.Queries))
	for _, q := range q.Queries {
		q = q.Optimize()
		if or, ok := q.(*Or); ok {
			qs = append(qs, or.Queries...)
		} else {
			qs = append(qs, q)
		}
	}
	return &Or{Queries: qs}
}

func (q *Or) invert() Query {
	qs := make([]Query, len(q.Queries))
	for i, q := range q.Queries {
		qs[i] = &Not{Query: q}
	}
	return (&And{Queries: qs}).Optimize()
}

func (q *Or) TimeRange() (*time.Time, *time.Time) {
	var start, end *time.Time

	// Get maximum range.
	// For example: "(time>=2025-01-01 time<=2025-12-31) OR (time>=2026-01-01 time<=2026-12-31)"  is the same as "time>=2025-01-01 time<=2026-12-31"
	for _, query := range q.Queries {
		s, e := query.TimeRange()
		if s != nil && (start == nil || s.Before(*start)) {
			start = s
		}
		if e != nil && (end == nil || e.After(*end)) {
			end = e
		}
	}

	if start != nil && end != nil && start.After(*end) {
		return nil, nil
	}

	return start, end
}

type Not struct {
	Query Query
}

func (q *Not) String() string {
	return "(NOT " + q.Query.String() + ")"
}

func (q *Not) Match(r api.Record) bool {
	return !q.Query.Match(r)
}

func (q *Not) Optimize() Query {
	q2 := q.Query.Optimize()

	if inverter, ok := q2.(queryInverter); ok {
		return inverter.invert()
	}

	return &Not{Query: q2}
}

func (q *Not) invert() Query {
	return q.Query
}

func (q *Not) TimeRange() (*time.Time, *time.Time) {
	start, end := q.Query.TimeRange()

	if start == nil && end == nil || start != nil && end != nil {
		return nil, nil
	}

	// When only have the start time like "not time>=2025-01-01", it means the same as only have the end time like "time<2025-01-01" in the query.
	if start != nil {
		e := start.Add(-1)
		return nil, &e
	}

	// When only have the end time like "not time<=2025-01-01", it means the same as only have the start time like "time>2025-01-01" in the query.
	if end != nil {
		s := end.Add(1)
		return &s, nil
	}

	// unreachable
	return nil, nil
}

type SimpleQuery struct {
	Value valueMatcher
}

func (q *SimpleQuery) String() string {
	return q.Value.String()
}

func (q *SimpleQuery) Match(r api.Record) bool {
	if q.Value.Match(r.Time) {
		return true
	}
	if q.Value.Match(r.Status) {
		return true
	}
	if q.Value.Match(r.Latency) {
		return true
	}
	if q.Value.Match(r.Target) {
		return true
	}
	if q.Value.Match(r.Message) {
		return true
	}
	return false
}

func (q *SimpleQuery) Optimize() Query {
	return q
}

func (q *SimpleQuery) TimeRange() (*time.Time, *time.Time) {
	if m, ok := q.Value.(timeValueMatcher); ok {
		return m.TimeRange()
	}

	return nil, nil
}

type FieldQuery struct {
	Key   stringMatcher
	Value valueMatcher
}

func (q *FieldQuery) String() string {
	return fmt.Sprintf("%q%s", q.Key, q.Value)
}

func (q *FieldQuery) Match(r api.Record) bool {
	if q.Key.Match("time") && q.Value.Match(r.Time) {
		return true
	}
	if q.Key.Match("status") && q.Value.Match(r.Status) {
		return true
	}
	if q.Key.Match("latency") && q.Value.Match(r.Latency) {
		return true
	}
	if q.Key.Match("target") && q.Value.Match(r.Target) {
		return true
	}
	if q.Key.Match("message") && q.Value.Match(r.Message) {
		return true
	}

	for key, value := range r.Extra {
		if q.Key.Match(key) {
			if vs, ok := value.([]any); ok {
				for _, v := range vs {
					if q.Value.Match(v) {
						return true
					}
				}
			} else if q.Value.Match(value) {
				return true
			}
		}
	}

	return false
}

func (q *FieldQuery) Optimize() Query {
	return q
}

func (q *FieldQuery) TimeRange() (*time.Time, *time.Time) {
	if !q.Key.Match("time") {
		return nil, nil
	}

	if m, ok := q.Value.(timeValueMatcher); ok {
		return m.TimeRange()
	}

	return nil, nil
}
