package query

import (
	"testing"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a", `(AND ="*a*")`},
		{"a b", `(AND ="*a*" ="*b*")`},
		{"a OR b", `(AND (OR ="*a*" ="*b*"))`},
		{"a b OR c d", `(AND ="*a*" (OR ="*b*" ="*c*") ="*d*")`},
		{"(a b) OR (c AND d)", `(AND (OR (AND ="*a*" ="*b*") (AND ="*c*" ="*d*")))`},
		{"((a b) OR c OR d) e", `(AND (OR (AND ="*a*" ="*b*") ="*c*" ="*d*") ="*e*")`},
		{"a OR b OR c OR d", `(AND (OR ="*a*" ="*b*" ="*c*" ="*d*"))`},
		{"a AND b c", `(AND ="*a*" ="*b*" ="*c*")`},
		{"!a -b NOT c", `(AND (NOT ="*a*") (NOT ="*b*") (NOT ="*c*"))`},
		{"!!a ---b NOT !-a", `(AND ="*a*" (NOT ="*b*") (NOT ="*a*"))`},
		{`"a b" c\ d`, `(AND ="*a b*" ="*c d*")`},
		{"*a b* *c* d*e *f*g*", `(AND ="*a*" ="*b*" ="*c*" ="*d*e*" ="*f*g*")`},
		{"=*a =b* =*c* =d*e =*f*g*", `(AND ="*a" ="b*" ="*c*" ="d*e" ="*f*g*")`},
		{"=a <>b >1 <2s >=2003-03-30T15:33Z <=4h", `(AND ="a" !="b" >1.000000 <2s >=2003-03-30T15:33:00Z <=4h0m0s)`},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got := ParseQuery(test.input)
			if got.String() != test.want {
				t.Errorf("unexpected result\n got %s\nwant %s", got, test.want)
			}
		})
	}
}
