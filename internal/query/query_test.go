package query

import (
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
		{"a b OR c d", `(AND ="*a*" (OR ="*b*" ="*c*") ="*d*")`},
		{"(a b) OR (c AND d)", `(OR (AND ="*a*" ="*b*") (AND ="*c*" ="*d*"))`},
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
