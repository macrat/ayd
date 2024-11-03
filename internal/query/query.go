package query

import (
	"fmt"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

var (
	minTime = time.Time{}
	maxTime = time.Unix(2<<31-1, 0)
)

type Query interface {
	fmt.Stringer

	Match(api.Record) bool
	Period() (time.Time, time.Time)
	Optimize() Query
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
			pushQuery(&SimpleQuery{Value: tok.Token().Value})
		case fieldKeywordToken:
			pushQuery(&FieldQuery{Key: tok.Token().Key, Value: tok.Token().Value})
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

func (q *And) Period() (time.Time, time.Time) {
	start := minTime
	end := maxTime
	for _, query := range q.Queries {
		s, e := query.Period()
		if s.After(start) {
			start = s
		}
		if e.Before(end) {
			end = e
		}
	}
	return start, end
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

func (q *Or) Period() (time.Time, time.Time) {
	start := maxTime
	end := minTime
	for _, query := range q.Queries {
		s, e := query.Period()
		if s.Before(start) {
			start = s
		}
		if e.After(end) {
			end = e
		}
	}
	return start, end
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

type Not struct {
	Query Query
}

func (q *Not) String() string {
	return "(NOT " + q.Query.String() + ")"
}

func (q *Not) Period() (time.Time, time.Time) {
	start, end := q.Query.Period()
	if start.IsZero() && !end.Equal(maxTime) {
		return end, maxTime
	}
	if !start.IsZero() && end.Equal(maxTime) {
		return minTime, start
	}
	return minTime, maxTime
}

func (q *Not) Match(r api.Record) bool {
	return !q.Query.Match(r)
}

func (q *Not) Optimize() Query {
	q2 := q.Query.Optimize()
	if not, ok := q2.(*Not); ok {
		return not.Query
	}
	return &Not{Query: q2}
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

func (q *SimpleQuery) Period() (time.Time, time.Time) {
	if t, ok := q.Value.(timeValueMatcher); ok {
		return t.Period()
	}
	return minTime, maxTime
}

func (q *SimpleQuery) Optimize() Query {
	return q
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
		if q.Key.Match(key) && q.Value.Match(value) {
			return true
		}
	}

	return false
}

func (q *FieldQuery) Period() (time.Time, time.Time) {
	if t, ok := q.Value.(timeValueMatcher); ok && q.Key.Match("time") {
		return t.Period()
	}
	return minTime, maxTime
}

func (q *FieldQuery) Optimize() Query {
	return q
}
