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
		{"=a <>b >1 <2s >=2003-03-30T15:33Z <=4h 5ms", `(AND ="a" (NOT ="b") >1.000000 <2s >=2003-03-30T15:33:00Z <=4h0m0s in5ms)`},
		{"<a >b <=c >=d ==e f<g", `(AND ="*<a*" ="*>b*" ="*<=c*" ="*>=d*" ="e" ="*f<g*")`},
		{"a=b c!=d e>1 f<2s g>=2003-03-30T15:33Z h<=4h", `(AND "a"="b" (NOT "c"="d") "e">1.000000 "f"<2s "g">=2003-03-30T15:33:00Z "h"<=4h0m0s)`},
		{"<a>b=<c d=e=f g!=h!=i j=k<>l", `(AND "<a>b"="<c" "d"="e=f" (NOT "g"="h!=i") "j"="k<>l")`},
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
		{`target=*://localhost*`, R{Target: "http://localhost:8080"}, true},
		{`target=*localhost*`, R{Target: "http://example.com", Extra: map[string]any{"non-target": "localhost"}}, false},
		{`=HEALTHY`, R{Message: "HEALTHY"}, true},
		{`=HEALTHY 1`, R{Status: lib.StatusHealthy, Message: "1"}, true},
		{`=HEALTHY 2`, R{Status: lib.StatusFailure, Message: "2"}, false},
		{`status=HEALTHY`, R{Status: lib.StatusHealthy, Message: "FAILURE"}, true},
		{`status=HEALTHY 1`, R{Status: lib.StatusHealthy, Message: "HEALTHY 1"}, true},
		{`status=HEALTHY 2`, R{Status: lib.StatusFailure, Message: "HEALTHY 2"}, false},
		{`=FAILURE 1`, R{Status: lib.StatusFailure, Message: "FAILURE 1"}, true},
		{`=FAILURE 2`, R{Status: lib.StatusHealthy, Message: "FAILURE 2"}, false},
		{`<>HEALTHY 1`, R{Status: lib.StatusHealthy, Message: "1"}, false},
		{`<>HEALTHY 2`, R{Status: lib.StatusFailure, Message: "2"}, true},
		{`!=HEALTHY 1`, R{Status: lib.StatusHealthy, Message: "1"}, false},
		{`!=HEALTHY 2`, R{Status: lib.StatusFailure, Message: "2"}, true},
		{`NOT =HEALTHY 1`, R{Status: lib.StatusHealthy, Message: "1"}, false},
		{`NOT =HEALTHY 2`, R{Status: lib.StatusFailure, Message: "2"}, true},
		{`status!=HEALTHY 1`, R{Status: lib.StatusHealthy, Message: "1"}, false},
		{`status!=HEALTHY 2`, R{Status: lib.StatusFailure, Message: "2"}, true},

		{`int=12*`, R{Extra: map[string]any{"int": 123}}, true},
		{`int=123*`, R{Extra: map[string]any{"int": 123}}, true},
		{`int=1234*`, R{Extra: map[string]any{"int": 123}}, false},

		{`float=1*.1*`, R{Extra: map[string]any{"float": 123.123}}, true},
		{`float=2*.2*`, R{Extra: map[string]any{"float": 123.123}}, false},

		{`array=hello`, R{Extra: map[string]any{"array": []any{"hello", "world"}}}, true},
		{`array=wor*`, R{Extra: map[string]any{"array": []any{"hello", "world"}}}, true},
		{`array=foo`, R{Extra: map[string]any{"array": []any{"hello", "world"}}}, false},
	})
}

func TestNumberValueMatcher(t *testing.T) {
	RunQueryTest(t, []QueryTest{
		{`int=1`, R{Extra: map[string]any{"int": 1}}, true},
		{`int=2`, R{Extra: map[string]any{"int": 1}}, false},
		{`int<3`, R{Extra: map[string]any{"int": 3}}, false},
		{`int<4`, R{Extra: map[string]any{"int": 3}}, true},
		{`int>5`, R{Extra: map[string]any{"int": 6}}, true},
		{`int>6`, R{Extra: map[string]any{"int": 6}}, false},
		{`int<=7`, R{Extra: map[string]any{"int": 8}}, false},
		{`int<=8`, R{Extra: map[string]any{"int": 8}}, true},
		{`int<=9`, R{Extra: map[string]any{"int": 8}}, true},
		{`int>=10`, R{Extra: map[string]any{"int": 11}}, true},
		{`int>=11`, R{Extra: map[string]any{"int": 11}}, true},
		{`int>=12`, R{Extra: map[string]any{"int": 11}}, false},
		{`int!=13`, R{Extra: map[string]any{"int": 14}}, true},
		{`int!=14`, R{Extra: map[string]any{"int": 14}}, false},
		{`int!=15`, R{Extra: map[string]any{"int": 14}}, true},

		{`float=1.5`, R{Extra: map[string]any{"float": 1.5}}, true},
		{`float=2.5`, R{Extra: map[string]any{"float": 1.5}}, false},
		{`float<3.5`, R{Extra: map[string]any{"float": 3.5}}, false},
		{`float<4.5`, R{Extra: map[string]any{"float": 3.5}}, true},
		{`float>5.5`, R{Extra: map[string]any{"float": 6.5}}, true},
		{`float>6.5`, R{Extra: map[string]any{"float": 6.5}}, false},
		{`float<=7.5`, R{Extra: map[string]any{"float": 8.5}}, false},
		{`float<=8.5`, R{Extra: map[string]any{"float": 8.5}}, true},
		{`float<=9.5`, R{Extra: map[string]any{"float": 8.5}}, true},
		{`float>=10.5`, R{Extra: map[string]any{"float": 11.5}}, true},
		{`float>=11.5`, R{Extra: map[string]any{"float": 11.5}}, true},
		{`float>=12.5`, R{Extra: map[string]any{"float": 11.5}}, false},
		{`float!=13.5`, R{Extra: map[string]any{"float": 14.5}}, true},
		{`float!=14.5`, R{Extra: map[string]any{"float": 14.5}}, false},
		{`float!=15.5`, R{Extra: map[string]any{"float": 14.5}}, true},

		{`str=1`, R{Extra: map[string]any{"str": "1"}}, true},
		{`str=2.5`, R{Extra: map[string]any{"str": "2.5"}}, true},
		{`3`, R{Message: "12345"}, true},
		{`4`, R{Message: "67890"}, false},
		{`str=5`, R{Extra: map[string]any{"str": "five"}}, false},

		{`latency=1`, R{Latency: time.Millisecond}, true},
		{`latency=2`, R{Latency: time.Millisecond}, false},
		{`latency<3`, R{Latency: time.Millisecond * 4}, false},
		{`latency<4`, R{Latency: time.Millisecond * 4}, false},
		{`latency<5`, R{Latency: time.Millisecond * 4}, true},
		{`latency<=6`, R{Latency: time.Millisecond * 7}, false},
		{`latency<=7`, R{Latency: time.Millisecond * 7}, true},
		{`latency<=8`, R{Latency: time.Millisecond * 7}, true},
		{`latency>9`, R{Latency: time.Millisecond * 10}, true},
		{`latency>10`, R{Latency: time.Millisecond * 10}, false},
		{`latency>11`, R{Latency: time.Millisecond * 10}, false},
		{`latency>=12`, R{Latency: time.Millisecond * 13}, true},
		{`latency>=13`, R{Latency: time.Millisecond * 13}, true},
		{`latency>=14`, R{Latency: time.Millisecond * 13}, false},

		{`time=123`, R{}, false},
	})
}

func TestTimeValueMatcher(t *testing.T) {
	RunQueryTest(t, []QueryTest{
		{`2006-01-02T15:04:05Z`, R{Time: "2006-01-02T15:04:05Z"}, true},
		{`2006-01-02T15:04:06Z`, R{Time: "2006-01-02T15:04:05Z"}, false},
		{`2000-01-01`, R{Time: "2000-01-01T00:00:00Z"}, true},
		{`2000-01-02`, R{Time: "2000-01-02T13:45:00Z"}, true},
		{`2000-01-03`, R{Time: "2000-01-03T23:59:59Z"}, true},
		{`2000-01-04`, R{Time: "2000-01-03T00:00:00Z"}, false},
		{`2000-01-05`, R{Time: "2000-01-06T00:00:00Z"}, false},
		{`=2000-01-01`, R{Time: "2000-01-01T00:00:00Z"}, true},
		{`=2000-01-02`, R{Time: "2000-01-02T23:59:59Z"}, true},
		{`=2024-12-31T15:04:06+09:00`, R{Time: "2024-12-31T06:04:06Z"}, true},
		{`time=2000-01-03`, R{Time: "2000-01-03T23:59:59Z"}, true},
		{`time=2000-01-04`, R{Time: "2000-01-03T00:00:00Z"}, false},

		{`<>2000-01-02`, R{Time: "2000-01-01T23:59:59Z"}, true},
		{`<>2000-01-03`, R{Time: "2000-01-03T00:00:00Z"}, false},
		{`<>2000-01-04`, R{Time: "2000-01-04T23:50:50Z"}, false},
		{`<>2000-01-05`, R{Time: "2000-01-06T00:00:00Z"}, true},

		{`<2000-01-01`, R{Time: "2000-01-02T23:59:59Z"}, false},
		{`<2000-01-02`, R{Time: "2000-01-02T23:59:59Z"}, false},
		{`<2000-01-03`, R{Time: "2000-01-02T23:59:59Z"}, true},

		{`<=2000-01-01`, R{Time: "2000-01-02T23:59:59Z"}, false},
		{`<=2000-01-02`, R{Time: "2000-01-02T23:59:59Z"}, true},
		{`<=2000-01-03`, R{Time: "2000-01-02T23:59:59Z"}, true},

		{`>2000-01-01`, R{Time: "2000-01-02T00:00:00Z"}, true},
		{`>2000-01-02`, R{Time: "2000-01-02T00:00:00Z"}, false},
		{`>2000-01-03`, R{Time: "2000-01-02T00:00:00Z"}, false},

		{`>=2000-01-01`, R{Time: "2000-01-02T00:00:00Z"}, true},
		{`>=2000-01-02`, R{Time: "2000-01-02T00:00:00Z"}, true},
		{`>=2000-01-03`, R{Time: "2000-01-02T00:00:00Z"}, false},

		{`>=2000-01-01 <=2000-01-01`, R{Time: "2000-01-01T00:00:00Z"}, true},
		{`>=2000-01-02 <=2000-01-02`, R{Time: "2000-01-01T00:00:00Z"}, false},

		{`str=2000-01-01`, R{Extra: map[string]any{"str": "2000-01-01T00:00:00Z"}}, true},
		{`str=2000-01-02`, R{Extra: map[string]any{"str": "2000-01-01T00:00:00Z"}}, false},
		{`str=2000-01-03`, R{Extra: map[string]any{"str": "2000-01-03"}}, true},
		{`str=2000-01-04`, R{Extra: map[string]any{"str": "2000-01-03"}}, false},
		{`2000-01-05`, R{Message: "today is 2000-01-05"}, true},
		{`str=2000-01-06`, R{Extra: map[string]any{"str": "not a date"}}, false},

		{`int=1970-01-01`, R{Extra: map[string]any{"int": 0}}, true},
		{`int=2024-01-01T00:00:00Z`, R{Extra: map[string]any{"int": 1704067200}}, true},
		{`float=1970-01-01`, R{Extra: map[string]any{"float": 0.1}}, true},
		{`float=2024-01-01T00:00:00Z`, R{Extra: map[string]any{"float": 1704067200.1}}, true},
	})
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
