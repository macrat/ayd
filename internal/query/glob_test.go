package query

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func ParseGlob(s string) stringMatcher {
	b := newGlobBuilder()
	for _, r := range s {
		b.Feed(r)
	}
	return b.Build()
}

func TestGlobBuilder(t *testing.T) {
	//t.Parallel()

	tests := []struct {
		input  string
		output stringMatcher
	}{
		{
			input: `foo*bar`,
			output: globMatcher{
				Tokens: []string{
					"foo",
				},
				MinLength: 6,
				Suffix:    "bar",
			},
		},
		{
			input: `foo\*bar`,
			output: exactMatcher{"foo*bar"},
		},
		{
			input: `*foo*bar*baz`,
			output: globMatcher{
				Tokens: []string{
					"",
					"foo",
					"bar",
				},
				MinLength: 9,
				Suffix:    "baz",
			},
		},
		{
			input: `foo*bar*`,
			output: globMatcher{
				Tokens: []string{
					"foo",
					"bar",
				},
				MinLength: 6,
				Suffix:    "",
			},
		},
		{
			input: `hello\nworld\tfoo\rbar`,
			output: exactMatcher{"hello\nworld\tfoo\rbar"},
		},
		{
			input: `foo***bar**`,
			output: globMatcher{
				Tokens: []string{
					"foo",
					"bar",
				},
				MinLength: 6,
				Suffix:    "",
			},
		},
		{
			input: `foobar*`,
			output: prefixMatcher{"foobar"},
		},
		{
			input: `*foobar`,
			output: suffixMatcher{"foobar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			g := ParseGlob(tt.input)
			if diff := cmp.Diff(tt.output, g); diff != "" {
				t.Fatalf("unexpected output (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		glob  string
		input string
		match bool
	}{
		{
			glob:  `foo*bar1`,
			input: `foobar1`,
			match: true,
		},
		{
			glob:  `foo*bar2`,
			input: `foo`,
			match: false,
		},
		{
			glob:  `foo*bar3`,
			input: `fooxxxbar3`,
			match: true,
		},
		{
			glob:  `foo*bar*bar*4`,
			input: `fooxxxbarxxxbarxxxbarxxx4`,
			match: true,
		},
		{
			glob:  `foo*bar*bar*5`,
			input: `fooxxxbarxxxbarxxxbarxxx`,
			match: false,
		},
		{
			glob:  `foobar6*`,
			input: `foobar6`,
			match: true,
		},
		{
			glob:  `foobar7*`,
			input: `foobar7xxx`,
			match: true,
		},
		{
			glob:  `*foobar8`,
			input: `xxxfoobar8`,
			match: true,
		},
		{
			glob:  `*foobar9`,
			input: `xxxfoobar9xxx`,
			match: false,
		},
		{
			glob:  `*foobar10`,
			input: `foobar10`,
			match: true,
		},
		{
			glob:  `foo\*bar\n11`,
			input: "foo*bar\n11",
			match: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.glob, func(t *testing.T) {
			g := ParseGlob(tt.glob)
			if g.Match(tt.input) != tt.match {
				t.Fatalf("%q.Match(%q) expected %v but got %#v\n%#v", tt.glob, tt.input, tt.match, !tt.match, g)
			}
		})
	}
}

func FuzzGlob(f *testing.F) {
	f.Add(`foo*bar`, "foobar")
	f.Add(`foo\*bar`, "fooxxxbar")
	f.Add(`*foo*bar`, "fooabcbar")
	f.Add(`foo*bar*`, "foo123barbaz")
	f.Add(`hello\nworld\tfoo\rbar`, "hello\nworld\tfoo\rbar")
	f.Add(`0*0*0`, "000000000000000000000000000001")
	f.Add(``, "")

	f.Fuzz(func(t *testing.T, pattern string, input string) {
		g := ParseGlob(pattern)
		g.Match(input)
	})
}

func BenchmarkGlobMatch(b *testing.B) {
	tests := []struct {
		glob   string
		regexp *regexp.Regexp
		input  string
	}{
		{
			glob:   `foo*bar1`,
			regexp: regexp.MustCompile(`^foo.*bar1$`),
			input:  `fooxxxxxxxxxbar1`,
		},
		{
			glob:   `foobar*barbaz2`,
			regexp: regexp.MustCompile(`^foo.*bar2$`),
			input:  `foobaaaabaz2`,
		},
		{
			glob:   `foobar3*`,
			regexp: regexp.MustCompile(`^foobar3.*$`),
			input:  `foobar3abcdefghijklmnopqrstuvwxyz`,
		},
		{
			glob:   `*foobar4`,
			regexp: regexp.MustCompile(`^foobar3.*$`),
			input:  `abcdefghijklmnopqrstuvwxyzfoobar4`,
		},
		{
			glob:   `0*0*0`,
			regexp: regexp.MustCompile(`^0.*0.*0$`),
			input:  `00000000000000000000x0`,
		},
		{
			glob:   `1*1*1`,
			regexp: regexp.MustCompile(`^1.*1.*1$`),
			input:  `11111111111111111111x1`,
		},
		{
			glob:   `2*2*2*`,
			regexp: regexp.MustCompile(`^2.*2.*2.*$`),
			input:  `22222222222222222222x2`,
		},
	}

	for _, tt := range tests {
		tt := tt
		b.Run(tt.glob, func(b *testing.B) {
			b.Run("glob", func(b *testing.B) {
				g := ParseGlob(tt.glob)

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					g.Match(tt.input)
				}
			})
			b.Run("regexp", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					tt.regexp.MatchString(tt.input)
				}
			})
		})
	}
}
