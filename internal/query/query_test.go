package query

import (
	"time"

	"testing"

	lib "github.com/macrat/ayd/lib-ayd"
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
		{"a OR b c d", `(OR ="*a*" (AND ="*b*" ="*c*" ="*d*"))`},
		{"!a -b NOT c", `(AND (NOT ="*a*") (NOT ="*b*") (NOT ="*c*"))`},
		{"!!a ---b NOT !-a", `(AND ="*a*" (NOT ="*b*") (NOT ="*a*"))`},
		{"!(!a)", `="*a*"`},
		{"a b \t", `(AND ="*a*" ="*b*")`},
		{`"a b" c\ d`, `(AND ="*a b*" ="*c d*")`},
		{"*a b* *c* d*e *f*g*", `(AND ="*a*" ="*b*" ="*c*" ="*d*e*" ="*f*g*")`},
		{"=*a =b* =*c* =d*e =*f*g*", `(AND ="*a" ="b*" ="*c*" ="d*e" ="*f*g*")`},
		{"=a <>b >1 <2s >=2003-03-30T15:33Z <=4h 5ms", `(AND ="a" !="b" >1.000000 <2s >=2003-03-30T15:33:00Z <=4h0m0s in5ms)`},
		{"<a >b <=c >=d ==e f<g", `(AND ="*<a*" ="*>b*" ="*<=c*" ="*>=d*" ="e" ="*f<g*")`},
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
		{"a=b", minTime, maxTime},
		{"<2024-01-02T15:04:05Z", minTime, time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC)},
		{">=2024-01-02T15:04:05Z", time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC), maxTime},
		{"=2024-01-02T15:04:05Z", time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC), time.Date(2024, 1, 2, 15, 4, 5, 999999999, time.UTC)},
		{">=2024-01-02 <2024-02-03", time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2024, 2, 3, 0, 0, 0, 0, time.UTC)},
		{">2024-01-02 <2024-02-03", time.Date(2024, 1, 2, 23, 59, 59, 999999999, time.UTC), time.Date(2024, 2, 3, 0, 0, 0, 0, time.UTC)},
		{">2024-01-01 >=2024-02-01", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), maxTime},
		{"<=2024-01-01 <2024-02-01", minTime, time.Date(2024, 1, 1, 23, 59, 59, 999999999, time.UTC)},
		{">=2024-01-01 <2024-02-01 OR >=2024-03-01 <2024-04-01", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)},
		{"<2024-01-01 OR <2024-02-01", minTime, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)},
		{">=2024-01-01 OR >=2024-02-01", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), maxTime},
		{"<2024-01-01 a", minTime, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"<2024-01-01 OR a", minTime, maxTime},
		{">=2024-01-01 a", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), maxTime},
		{">=2024-01-01 OR a", minTime, maxTime},
		{"-2024-01-01", minTime, maxTime},
		{"<>2024-01-01", minTime, maxTime},
		{"NOT (<2024-01-01)", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), maxTime},
		{"NOT (>=2024-01-01)", minTime, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"NOT (>=2024-01-01 <2024-02-01)", minTime, maxTime},
		{"time<2024-01-01", minTime, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"time>=2024-01-01", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), maxTime},
		{"time=2024-01-01", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 23, 59, 59, 999999999, time.UTC)},
		{"time!=2024-01-01", minTime, maxTime},
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

type R struct {
	Time    string
	Status  lib.Status
	Latency time.Duration
	Target  string
	Message string
	Extra   map[string]any
}

type QueryTest struct {
	Query  string
	Record R
	Result bool
}

func RunQueryTest(t *testing.T, tests []QueryTest) {
	t.Helper()

	RtoRecord := func(r R) lib.Record {
		t.Helper()

		if r.Time == "" {
			r.Time = "2006-01-02T15:04:05Z"
		}
		ts, err := lib.ParseTime(r.Time)
		if err != nil {
			t.Fatalf("failed to parse time: %s", err)
		}

		if r.Target == "" {
			r.Target = "dummy:"
		}
		u, err := lib.ParseURL(r.Target)
		if err != nil {
			t.Fatalf("failed to parse target: %s", err)
		}

		return lib.Record{
			Time:    ts,
			Status:  r.Status,
			Latency: r.Latency,
			Target:  u,
			Message: r.Message,
			Extra:   r.Extra,
		}
	}

	for _, tt := range tests {
		t.Run(tt.Query, func(t *testing.T) {
			q := ParseQuery(tt.Query)
			rec := RtoRecord(tt.Record)
			if q.Match(rec) != tt.Result {
				t.Errorf("unexpected result for %q wanted %v.\nrecord: %s\nparsed query: %s", tt.Query, tt.Result, rec, q)
			}
		})
	}
}

func TestStringValueMatcher(t *testing.T) {
	RunQueryTest(t, []QueryTest{
		{`hello`, R{Message: "hello world"}, true},
		{`=hello`, R{Message: "hello world"}, false},
		{`=hello*`, R{Message: "hello world"}, true},
		{`=*world`, R{Message: "hello world"}, true},
		{`=*o\ w*`, R{Message: "hello world"}, true},
		{`message=hello\ world`, R{Message: "hello world"}, true},
		{`message=foobar`, R{Target: "dummy:foobar"}, false},
		{`http*://example.com/1`, R{Target: "https://example.com/1"}, true},
		{`http*://example.com/2`, R{Target: "http://example.com/2"}, true},
		{`http*://example.com/3`, R{Target: "https://www.example.com/3"}, false},
		{`=HEALTHY`, R{Message: "HEALTHY"}, true},
		{`=HEALTHY 2`, R{Status: lib.StatusHealthy, Message: "2"}, true},
		{`status=HEALTHY`, R{Status: lib.StatusHealthy, Message: "FAILURE"}, true},
		{`status=HEALTHY 1`, R{Status: lib.StatusHealthy, Message: "HEALTHY 1"}, true},
		{`status=HEALTHY 2`, R{Status: lib.StatusFailure, Message: "HEALTHY 2"}, false},
		{`=FAILURE 1`, R{Status: lib.StatusFailure, Message: "FAILURE 1"}, true},
		{`=FAILURE 2`, R{Status: lib.StatusHealthy, Message: "FAILURE 2"}, false},
	})
}

func TestNumberValueMatcher(t *testing.T) {
	t.Skip("not implemented yet")
	RunQueryTest(t, []QueryTest{})
}

func TestTimeValueMatcher(t *testing.T) {
	t.Skip("not implemented yet")
	RunQueryTest(t, []QueryTest{})
}

func TestDurationValueMatcher(t *testing.T) {
	RunQueryTest(t, []QueryTest{
		{`>0s`, R{Latency: time.Second}, true},
		{`>1s`, R{Latency: time.Second}, false},
		{`>2s`, R{Latency: time.Second}, false},
		{`>=0s`, R{Latency: time.Second}, true},
		{`>=1s`, R{Latency: time.Second}, true},
		{`>=2s`, R{Latency: time.Second}, false},
		{`<0s`, R{Latency: time.Second}, false},
		{`<1s`, R{Latency: time.Second}, false},
		{`<2s`, R{Latency: time.Second}, true},
		{`<=0s`, R{Latency: time.Second}, false},
		{`<=1s`, R{Latency: time.Second}, true},
		{`<=2s`, R{Latency: time.Second}, true},
		{`=0s`, R{Latency: time.Second}, false},
		{`=1s`, R{Latency: time.Second}, true},
		{`=2s`, R{Latency: time.Second}, false},
		{`<>0s`, R{Latency: time.Second}, true},
		{`<>1s`, R{Latency: time.Second}, false},
		{`<>2s`, R{Latency: time.Second}, true},
		{`0s`, R{Latency: time.Second}, false},
		{`1s`, R{Latency: time.Second}, true},
		{`2s`, R{Latency: time.Second}, false},

		{`1ms`, R{Message: "4321ms"}, true},
		{`str=1s`, R{Extra: map[string]any{"str": "1000ms"}}, true},
		{`str=2s`, R{Extra: map[string]any{"str": "1000ms"}}, false},
		{`int=1.234s`, R{Extra: map[string]any{"int": 1234}}, true},
		{`int=5.678s`, R{Extra: map[string]any{"int": 56789}}, false},
		{`float=12340us`, R{Extra: map[string]any{"float": 12.34}}, true},
		{`float=23450us`, R{Extra: map[string]any{"float": 12.34}}, false},
	})
}
