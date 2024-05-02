package query

import (
	"testing"
	//	"github.com/google/go-cmp/cmp"
)

/*
func TestTokenizer(t *testing.T) {
	tests := []struct {
		Query string
		Want  []token
	}{
		{
			Query: "a and b",
			Want: []token{
				{Type: atomToken, Value: &atomValue{Right: "a"}},
				{Type: atomToken, Value: &atomValue{Right: "b"}},
			},
		},
		{
			Query: "hello (world OR foo) AND -bar",
			Want: []token{
				{Type: atomToken, Value: &atomValue{Right: "hello"}},
				{Type: lparenToken},
				{Type: atomToken, Value: &atomValue{Right: "world"}},
				{Type: orToken},
				{Type: atomToken, Value: &atomValue{Right: "foo"}},
				{Type: rparenToken},
				{Type: notToken},
				{Type: atomToken, Value: &atomValue{Right: "bar"}},
			},
		},
		{
			Query: "-- not ((foo)bar)",
			Want: []token{
				{Type: notToken},
				{Type: notToken},
				{Type: notToken},
				{Type: lparenToken},
				{Type: lparenToken},
				{Type: atomToken, Value: &atomValue{Right: "foo"}},
				{Type: rparenToken},
				{Type: atomToken, Value: &atomValue{Right: "bar"}},
				{Type: rparenToken},
			},
		},
		{
			Query: "o n orange nothing",
			Want: []token{
				{Type: atomToken, Value: &atomValue{Right: "o"}},
				{Type: atomToken, Value: &atomValue{Right: "n"}},
				{Type: atomToken, Value: &atomValue{Right: "orange"}},
				{Type: atomToken, Value: &atomValue{Right: "nothing"}},
			},
		},
		{
			Query: "hello=world foo==bar abc<def ghi>=jkl mno!=pqr stu<>vwx",
			Want: []token{
				{Type: atomToken, Value: &atomValue{Left: "hello", Op: opEqual, Right: "world"}},
				{Type: atomToken, Value: &atomValue{Left: "foo", Op: opEqual, Right: "bar"}},
				{Type: atomToken, Value: &atomValue{Left: "abc", Op: opLessThan, Right: "def"}},
				{Type: atomToken, Value: &atomValue{Left: "ghi", Op: opGreaterEqual, Right: "jkl"}},
				{Type: atomToken, Value: &atomValue{Left: "mno", Op: opNotEqual, Right: "pqr"}},
				{Type: atomToken, Value: &atomValue{Left: "stu", Op: opNotEqual, Right: "vwx"}},
			},
		},
		{
			Query: "foo=bar=baz a>b c<d e>=f g<=h i<>j k!=l m==n",
			Want: []token{
				{Type: atomToken, Value: &atomValue{Left: "foo", Op: opEqual, Right: "bar=baz"}},
				{Type: atomToken, Value: &atomValue{Left: "a", Op: opGreaterThan, Right: "b"}},
				{Type: atomToken, Value: &atomValue{Left: "c", Op: opLessThan, Right: "d"}},
				{Type: atomToken, Value: &atomValue{Left: "e", Op: opGreaterEqual, Right: "f"}},
				{Type: atomToken, Value: &atomValue{Left: "g", Op: opLessEqual, Right: "h"}},
				{Type: atomToken, Value: &atomValue{Left: "i", Op: opNotEqual, Right: "j"}},
				{Type: atomToken, Value: &atomValue{Left: "k", Op: opNotEqual, Right: "l"}},
				{Type: atomToken, Value: &atomValue{Left: "m", Op: opEqual, Right: "n"}},
			},
		},
		{
			Query: "<a >b =c !=d <=e >=f <>g ==h == <",
			Want: []token{
				{Type: atomToken, Value: &atomValue{Op: opLessThan, Right: "a"}},
				{Type: atomToken, Value: &atomValue{Op: opGreaterThan, Right: "b"}},
				{Type: atomToken, Value: &atomValue{Op: opEqual, Right: "c"}},
				{Type: notToken},
				{Type: atomToken, Value: &atomValue{Op: opEqual, Right: "d"}},
				{Type: atomToken, Value: &atomValue{Op: opLessEqual, Right: "e"}},
				{Type: atomToken, Value: &atomValue{Op: opGreaterEqual, Right: "f"}},
				{Type: atomToken, Value: &atomValue{Op: opNotEqual, Right: "g"}},
				{Type: atomToken, Value: &atomValue{Op: opEqual, Right: "h"}},
				{Type: atomToken, Value: &atomValue{Right: "=="}},
				{Type: atomToken, Value: &atomValue{Right: "<"}},
			},
		},
		{
			Query: `hello \and world foo\ bar \(baz\)`,
			Want: []token{
				{Type: atomToken, Value: &atomValue{Right: "hello"}},
				{Type: atomToken, Value: &atomValue{Right: "and"}},
				{Type: atomToken, Value: &atomValue{Right: "world"}},
				{Type: atomToken, Value: &atomValue{Right: "foo bar"}},
				{Type: atomToken, Value: &atomValue{Right: "(baz)"}},
			},
		},
		{
			Query: `hello "world foo" bar "baz`,
			Want: []token{
				{Type: atomToken, Value: &atomValue{Right: "hello"}},
				{Type: atomToken, Value: &atomValue{Right: "world foo"}},
				{Type: atomToken, Value: &atomValue{Right: "bar"}},
				{Type: atomToken, Value: &atomValue{Right: "baz"}},
			},
		},
		{
			Query: `hello "world\"foo" bar "(baz)"`,
			Want: []token{
				{Type: atomToken, Value: &atomValue{Right: "hello"}},
				{Type: atomToken, Value: &atomValue{Right: `world"foo`}},
				{Type: atomToken, Value: &atomValue{Right: "bar"}},
				{Type: atomToken, Value: &atomValue{Right: "(baz)"}},
			},
		},
		{
			Query: `hello" "world"("`,
			Want: []token{
				{Type: atomToken, Value: &atomValue{Right: "hello world("}},
			},
		},
		{
			Query: `hello\nworld foo\rbar abc\tdef`,
			Want: []token{
				{Type: atomToken, Value: &atomValue{Right: "hello\nworld"}},
				{Type: atomToken, Value: &atomValue{Right: "foo\rbar"}},
				{Type: atomToken, Value: &atomValue{Right: "abc\tdef"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Query, func(t *testing.T) {
			tok := newTokenizer(test.Query)
			var got []token
			for tok.Scan() {
				got = append(got, tok.token())
			}
			if diff := cmp.Diff(test.Want, got); diff != "" {
				t.Errorf("(-want, +got)\n%s", diff)
			}
		})
	}
}
*/

func TestTokenizer(t *testing.T) {
	tok := newTokenizer("hello=world")
	for tok.Scan() {
		t.Logf("%#v", tok.Token())
	}
}
