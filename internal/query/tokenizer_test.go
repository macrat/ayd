package query

import (
	"github.com/google/go-cmp/cmp"

	"testing"
)

func SimpleKeyword(s []string, op operator) token {
	ss := make([]*string, len(s))
	for x := range s {
		if s[x] != "" {
			ss[x] = &s[x]
		}
	}
	not := false
	if op == opNotEqual {
		not = true
		op = opEqual
	}
	return token{
		Type:  simpleKeywordToken,
		Not:   not,
		Value: newValueMatcher(ss, op),
	}
}

func FieldKeyword(field []string, op operator, value []string) token {
	ff := make([]*string, len(field))
	for x := range field {
		if field[x] != "" {
			ff[x] = &field[x]
		}
	}
	vv := make([]*string, len(value))
	for x := range value {
		if value[x] != "" {
			vv[x] = &value[x]
		}
	}
	not := false
	if op == opNotEqual {
		not = true
		op = opEqual
	}
	return token{
		Type:  fieldKeywordToken,
		Key:   newStringMatcher(ff),
		Not:   not,
		Value: newValueMatcher(vv, op),
	}
}

var (
	LPAREN = token{Type: lparenToken}
	RPAREN = token{Type: rparenToken}
	OR     = token{Type: orToken}
	NOT    = token{Type: notToken}
)

func TestTokenizer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Query string
		Want  []token
	}{
		{
			Query: "a and b",
			Want: []token{
				SimpleKeyword([]string{"a"}, opIncludes),
				SimpleKeyword([]string{"b"}, opIncludes),
			},
		},
		{
			Query: "hello (world OR foo) AND -bar",
			Want: []token{
				SimpleKeyword([]string{"hello"}, opIncludes),
				LPAREN,
				SimpleKeyword([]string{"world"}, opIncludes),
				OR,
				SimpleKeyword([]string{"foo"}, opIncludes),
				RPAREN,
				NOT,
				SimpleKeyword([]string{"bar"}, opIncludes),
			},
		},
		{
			Query: "-- not ((foo)bar)",
			Want: []token{
				NOT,
				NOT,
				NOT,
				LPAREN,
				LPAREN,
				SimpleKeyword([]string{"foo"}, opIncludes),
				RPAREN,
				SimpleKeyword([]string{"bar"}, opIncludes),
				RPAREN,
			},
		},
		{
			Query: "o n orange nothing",
			Want: []token{
				SimpleKeyword([]string{"o"}, opIncludes),
				SimpleKeyword([]string{"n"}, opIncludes),
				SimpleKeyword([]string{"orange"}, opIncludes),
				SimpleKeyword([]string{"nothing"}, opIncludes),
			},
		},
		{
			Query: "hello=world foo==bar abc<123 def>=456 ghi!=jkl mno<>pqr",
			Want: []token{
				FieldKeyword([]string{"hello"}, opEqual, []string{"world"}),
				FieldKeyword([]string{"foo"}, opEqual, []string{"bar"}),
				FieldKeyword([]string{"abc"}, opLessThan, []string{"123"}),
				FieldKeyword([]string{"def"}, opGreaterEqual, []string{"456"}),
				FieldKeyword([]string{"ghi"}, opNotEqual, []string{"jkl"}),
				FieldKeyword([]string{"mno"}, opNotEqual, []string{"pqr"}),
			},
		},
		{
			Query: "foo=bar=baz a>1s b<2m c>=2003-03-30T15:03Z d<=4 e<>f g!=h i==j",
			Want: []token{
				FieldKeyword([]string{"foo"}, opEqual, []string{"bar=baz"}),
				FieldKeyword([]string{"a"}, opGreaterThan, []string{"1s"}),
				FieldKeyword([]string{"b"}, opLessThan, []string{"2m"}),
				FieldKeyword([]string{"c"}, opGreaterEqual, []string{"2003-03-30T15:03Z"}),
				FieldKeyword([]string{"d"}, opLessEqual, []string{"4"}),
				FieldKeyword([]string{"e"}, opNotEqual, []string{"f"}),
				FieldKeyword([]string{"g"}, opNotEqual, []string{"h"}),
				FieldKeyword([]string{"i"}, opEqual, []string{"j"}),
			},
		},
		{
			Query: "<a >b =c !=d <>e f= g< <1ms >2002-02-20T14:02Z <=3s >=4 <>5 ==6.7 == <",
			Want: []token{
				SimpleKeyword([]string{"<a"}, opIncludes),
				SimpleKeyword([]string{">b"}, opIncludes),
				SimpleKeyword([]string{"c"}, opEqual),
				NOT,
				SimpleKeyword([]string{"d"}, opEqual),
				SimpleKeyword([]string{"e"}, opNotEqual),
				FieldKeyword([]string{"f"}, opEqual, []string{}),
				SimpleKeyword([]string{"g<"}, opIncludes),
				SimpleKeyword([]string{"1ms"}, opLessThan),
				SimpleKeyword([]string{"2002-02-20T14:02Z"}, opGreaterThan),
				SimpleKeyword([]string{"3s"}, opLessEqual),
				SimpleKeyword([]string{"4"}, opGreaterEqual),
				SimpleKeyword([]string{"5"}, opNotEqual),
				SimpleKeyword([]string{"6.7"}, opEqual),
				SimpleKeyword([]string{"=="}, opIncludes),
				SimpleKeyword([]string{"<"}, opIncludes),
			},
		},
		{
			Query: `hello \and world foo\ bar \(baz\)`,
			Want: []token{
				SimpleKeyword([]string{"hello"}, opIncludes),
				SimpleKeyword([]string{"and"}, opIncludes),
				SimpleKeyword([]string{"world"}, opIncludes),
				SimpleKeyword([]string{"foo bar"}, opIncludes),
				SimpleKeyword([]string{"(baz)"}, opIncludes),
			},
		},
		{
			Query: `hello "world foo" bar "baz`,
			Want: []token{
				SimpleKeyword([]string{"hello"}, opIncludes),
				SimpleKeyword([]string{"world foo"}, opIncludes),
				SimpleKeyword([]string{"bar"}, opIncludes),
				SimpleKeyword([]string{"baz"}, opIncludes),
			},
		},
		{
			Query: `hello "world\"foo" bar "(baz)"`,
			Want: []token{
				SimpleKeyword([]string{"hello"}, opIncludes),
				SimpleKeyword([]string{`world"foo`}, opIncludes),
				SimpleKeyword([]string{"bar"}, opIncludes),
				SimpleKeyword([]string{"(baz)"}, opIncludes),
			},
		},
		{
			Query: `hello" "world"("`,
			Want: []token{
				SimpleKeyword([]string{"hello world("}, opIncludes),
			},
		},
		{
			Query: `hello\nworld foo\rbar abc\tdef`,
			Want: []token{
				SimpleKeyword([]string{"hello\nworld"}, opIncludes),
				SimpleKeyword([]string{"foo\rbar"}, opIncludes),
				SimpleKeyword([]string{"abc\tdef"}, opIncludes),
			},
		},
		{
			Query: `this<is>key=and>>this<<is==value`,
			Want: []token{
				FieldKeyword([]string{"this<is>key"}, opEqual, []string{"and>>this<<is==value"}),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Query, func(t *testing.T) {
			tok := newTokenizer(test.Query)
			var got []token
			for tok.Scan() {
				got = append(got, tok.Token())
			}
			if diff := cmp.Diff(test.Want, got); diff != "" {
				t.Errorf("(-want, +got)\n%s", diff)
			}
		})
	}
}
