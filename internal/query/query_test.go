package query

import (
	"time"

	"testing"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a", `="*a*"`},
		{"a b", `(AND ="*a*" ="*b*")`},
		{"a OR b", `(OR ="*a*" ="*b*")`},
		{"a b OR c d", `(OR (AND ="*a*" ="*b*") (AND ="*c*" ="*d*"))`},
		{"(a b) OR (c AND d)", `(OR (AND ="*a*" ="*b*") (AND ="*c*" ="*d*"))`},
		{"a (b OR c) d", `(AND ="*a*" (OR ="*b*" ="*c*") ="*d*")`},
		{"(a b OR c) d", `(AND (OR (AND ="*a*" ="*b*") ="*c*") ="*d*")`},
		{"a (b OR c d)", `(AND ="*a*" (OR ="*b*" (AND ="*c*" ="*d*")))`},
		{"((a b) OR c OR d) e", `(AND (OR (AND ="*a*" ="*b*") ="*c*" ="*d*") ="*e*")`},
		{"a OR b OR c OR d", `(OR ="*a*" ="*b*" ="*c*" ="*d*")`},
		{"(a b) c d", `(AND ="*a*" ="*b*" ="*c*" ="*d*")`},
		{"a (b c) d", `(AND ="*a*" ="*b*" ="*c*" ="*d*")`},
		{"a b (c d)", `(AND ="*a*" ="*b*" ="*c*" ="*d*")`},
		{"a OR b c", `(OR ="*a*" (AND ="*b*" ="*c*"))`},
		{"(a))", `="*a*"`},
		{"a b) OR c", `(OR (AND ="*a*" ="*b*") ="*c*")`},
		{"a OR b) c", `(AND (OR ="*a*" ="*b*") ="*c*")`},
		{"a AND b c", `(AND ="*a*" ="*b*" ="*c*")`},
		{"!a -b NOT c", `(AND (NOT ="*a*") (NOT ="*b*") (NOT ="*c*"))`},
		{"!!a ---b NOT !-a", `(AND ="*a*" (NOT ="*b*") (NOT ="*a*"))`},
		{`"a b" c\ d`, `(AND ="*a b*" ="*c d*")`},
		{"*a b* *c* d*e *f*g*", `(AND ="*a*" ="*b*" ="*c*" ="*d*e*" ="*f*g*")`},
		{"=*a =b* =*c* =d*e =*f*g*", `(AND ="*a" ="b*" ="*c*" ="d*e" ="*f*g*")`},
		{"=a <>b >1 <2s >=2003-03-30T15:33Z <=4h", `(AND ="a" !="b" >1.000000 <2s >=2003-03-30T15:33:00Z <=4h0m0s)`},
		{"<a >b <=c >=d", `(AND ="*<a*" ="*>b*" ="*<=c*" ="*>=d*")`},
		{"a=b c!=d e>1 f<2s g>=2003-03-30T15:33Z h<=4h", `(AND "a"="b" "c"!="d" "e">1.000000 "f"<2s "g">=2003-03-30T15:33:00Z "h"<=4h0m0s)`},
		{"<a>b=<c d=e=f g!=h!=i j=k<>l", `(AND "<a>b"="<c" "d"="e=f" "g"!="h!=i" "j"="k<>l")`},
		{"target=https://example.com latency>1s", `(AND "target"="https://example.com" "latency">1s)`},
		{"HEALTHY <100ms", `(AND ="*HEALTHY*" <100ms)`},
		{"", `(AND)`},
		{`"`, `ANY`},
		{`""`, `ANY`},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got := ParseQuery(test.input)
			if got.String() != test.want {
				t.Errorf("unexpected result for %q\n got %s\nwant %s", test.input, got, test.want)
			}
		})
	}
}

func FuzzParseQuery(f *testing.F) {
	f.Add("a b")
	f.Add("a OR b")
	f.Add("a b OR c d")
	f.Add("(a b) OR (c AND d)")
	f.Add("a (b OR c) d")
	f.Add("(a b OR c) d")
	f.Add("a (b OR c d)")
	f.Add("((a b) OR c OR d) e")
	f.Add("a OR b OR c OR d")
	f.Add("(a b) c d")
	f.Add("a (b c) d")
	f.Add("a b (c d)")
	f.Add("a OR b c")
	f.Add("(a))")
	f.Add("a b) OR c")
	f.Add("a OR b) c")
	f.Add("a AND b c")
	f.Add("!a -b NOT c")
	f.Add("!!a ---b NOT !-a")
	f.Add(`"a b" c\ d`)
	f.Add("*a b* *c* d*e *f*g*")
	f.Add("=*a =b* =*c* =d*e =*f*g*")
	f.Add("=a <>b >1 <2s >=2003-03-30T15:33Z <=4h")
	f.Add("<a >b <=c >=d")
	f.Add("a=b c!=d e>1 f<2s g>=2003-03-30T15:33Z h<=4h")
	f.Add("<a>b=<c d=e=f g!=h!=i j=k<>l")
	f.Add("target=https://example.com latency>1s")
	f.Add("HEALTHY <100ms")

	f.Fuzz(func(t *testing.T, input string) {
		ParseQuery(input)
	})
}

func TestQuery_Period(t *testing.T) {
	minTime := time.Time{}
	maxTime := time.Unix(2<<31-1, 0)

	tests := []struct {
		query string
		start time.Time
		end   time.Time
	}{
		{"a", minTime, maxTime},
		{"a b", minTime, maxTime},
		{"<2024-01-02T15:04:05Z", minTime, time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC)},
		{">=2024-01-02T15:04:05Z", time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC), maxTime},
		{"=2024-01-02T15:04:05Z", time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC), time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC)},
		{">=2024-01-02 <2024-02-03", time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2024, 2, 3, 0, 0, 0, 0, time.UTC)},
		{">2024-01-02 <2024-02-03", time.Date(2024, 1, 2, 23, 59, 59, 999999999, time.UTC), time.Date(2024, 2, 3, 0, 0, 0, 0, time.UTC)},
		{">2024-01-01 >=2024-02-01", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), maxTime},
		{"<=2024-01-01 <2024-02-01", minTime, time.Date(2024, 1, 1, 23, 59, 59, 999999999, time.UTC)},
		{">=2024-01-01 <2024-02-01 OR >=2024-03-01 <2024-04-01", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)},
		{"<2024-01-01 a", minTime, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"<2024-01-01 OR a", minTime, maxTime},
	}

	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			q := ParseQuery(test.query)
			start, end := q.Period()
			if start != test.start || end != test.end {
				t.Errorf("unexpected result for %q\n got %s >>> %s\nwant %s >>> %s", test.query, start, end, test.start, test.end)
			}
		})
	}
}
