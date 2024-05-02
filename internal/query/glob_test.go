package query

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func ParseGlob(pattern []string) stringMatcher {
	b := &globBuilder{}
	for _, s := range pattern {
		if s == "" {
			b.FeedStar()
		} else {
			b.FeedLiteral(s)
		}
	}
	return b.Build()
}

func Pattern2String(pattern []string) string {
	xs := make([]string, len(pattern))
	for i, x := range pattern {
		if x == "" {
			xs[i] = "*"
		} else {
			xs[i] = fmt.Sprintf("%q", x)
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(xs, "_"))
}

func TestGlobBuilder(t *testing.T) {
	//t.Parallel()

	tests := []struct {
		input  []string
		output stringMatcher
	}{
		{
			input: []string{"foo", "", "bar"},
			output: globMatcher{
				Prefix:       "foo",
				Chunks:       []string{},
				Suffix:       "bar",
				ChunksLength: 0,
			},
		},
		{
			input:  []string{"foo*bar"},
			output: exactMatcher{"foo*bar"},
		},
		{
			input: []string{"", "foo", "", "bar", "", "baz"},
			output: globMatcher{
				Prefix: "",
				Chunks: []string{
					"foo",
					"bar",
				},
				Suffix:       "baz",
				ChunksLength: 6,
			},
		},
		{
			input: []string{"foo", "", "bar", ""},
			output: globMatcher{
				Prefix: "foo",
				Chunks: []string{
					"bar",
				},
				Suffix:       "",
				ChunksLength: 3,
			},
		},
		{
			input:  []string{"hello\nworld\tfoo\rbar"},
			output: exactMatcher{"hello\nworld\tfoo\rbar"},
		},
		{
			input: []string{"foo", "", "", "", "bar", "", ""},
			output: globMatcher{
				Prefix: "foo",
				Chunks: []string{
					"bar",
				},
				Suffix:       "",
				ChunksLength: 3,
			},
		},
		{
			input: []string{"", "foo", "", "bar", ""},
			output: globMatcher{
				Prefix: "",
				Chunks: []string{
					"foo",
					"bar",
				},
				Suffix:       "",
				ChunksLength: 6,
			},
		},
		{
			input: []string{""},
			output: globMatcher{
				Prefix:       "",
				Chunks:       nil,
				Suffix:       "",
				ChunksLength: 0,
			},
		},
		{
			input:  []string{},
			output: exactMatcher{""},
		},
		{
			input:  []string{"foobar", ""},
			output: prefixMatcher{"foobar"},
		},
		{
			input:  []string{"", "foobar"},
			output: suffixMatcher{"foobar"},
		},
		{
			input:  []string{"foobar"},
			output: exactMatcher{"foobar"},
		},
	}

	for _, tt := range tests {
		t.Run(Pattern2String(tt.input), func(t *testing.T) {
			g := ParseGlob(tt.input)
			if diff := cmp.Diff(tt.output, g); diff != "" {
				t.Fatalf("unexpected output (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		glob  []string
		input string
		match bool
	}{
		{
			glob:  []string{"foo", "", "bar1"},
			input: `foobar1`,
			match: true,
		},
		{
			glob:  []string{"foo", "", "bar2"},
			input: `foo`,
			match: false,
		},
		{
			glob:  []string{"foo", "", "bar3"},
			input: `fooxxxbar3`,
			match: true,
		},
		{
			glob:  []string{"foo", "", "bar", "", "bar", "", "bar", "", "4"},
			input: `fooxxxbarxxxbarxxxbarxxx4`,
			match: true,
		},
		{
			glob:  []string{"foo", "", "bar", "", "bar", "", "5"},
			input: `fooxxxbarxxxbarxxxbarxxx`,
			match: false,
		},
		{
			glob:  []string{"foobar6", ""},
			input: `foobar6`,
			match: true,
		},
		{
			glob:  []string{"foobar7", ""},
			input: `foobar7xxx`,
			match: true,
		},
		{
			glob:  []string{"", "foobar8"},
			input: `xxxfoobar8`,
			match: true,
		},
		{
			glob:  []string{"", "foobar9"},
			input: `xxxfoobar9xxx`,
			match: false,
		},
		{
			glob:  []string{"", "foobar10"},
			input: `foobar10`,
			match: true,
		},
		{
			glob:  []string{"foo*bar\n11"},
			input: "foo*bar\n11",
			match: true,
		},
		{
			glob:  []string{"", "foo", "", "bar", "", "12", ""},
			input: "helloxfoo-bar12xworld",
			match: true,
		},
		{
			glob:  []string{"", "foo", "", "bar", "", "13", ""},
			input: "helloxfoo-",
			match: false,
		},
		{
			glob:  []string{"", "foo", "", "bar", "", "14", ""},
			input: "helloxfoo",
			match: false,
		},
		{
			glob:  []string{"foo", "", "bar", "", "baz", "", "hello", "", "world", "", "15"},
			input: "foobarbazhelloworld15",
			match: true,
		},
	}

	for _, tt := range tests {
		t.Run(Pattern2String(tt.glob), func(t *testing.T) {
			g := ParseGlob(tt.glob)
			if g.Match(tt.input) != tt.match {
				t.Fatalf("%q.Match(%q) expected %v but got %#v\n%#v", tt.glob, tt.input, tt.match, !tt.match, g)
			}
		})
	}
}

/*
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
		g.Match(input)
		g.Match(input)
	})
}
*/

func BenchmarkGlobMatch(b *testing.B) {
	tests := []struct {
		name   string
		glob   []string
		regexp *regexp.Regexp
		input  string
	}{
		{
			name:   `foo*bar1`,
			glob:   []string{"foo", "", "bar1"},
			regexp: regexp.MustCompile(`^foo.*bar1$`),
			input:  `fooxxxxxxxxxxxxxxxxxxxxxxxxxbar1`,
		},
		{
			name:   `foobar*barbaz2`,
			glob:   []string{"foobar", "", "barbaz2"},
			regexp: regexp.MustCompile(`^foo.*bar2$`),
			input:  `foobaaaaaaaaaaaaaaaaaaaaaaaabaz2`,
		},
		{
			name:   `foobar*3*`,
			glob:   []string{"foobar", "", "3", ""},
			regexp: regexp.MustCompile(`^foobar.*3.*$`),
			input:  `foobar3abcdefghijklmnopqrstuvwxyz`,
		},
		{
			name:   `*foobar*4`,
			glob:   []string{"", "foobar", "", "4"},
			regexp: regexp.MustCompile(`^foobar.*4.*$`),
			input:  `abcdefghijklmnopqrstuvwxyzfoobar4`,
		},
		{
			name:   `0*0*0`,
			glob:   []string{"0", "", "0", "", "0"},
			regexp: regexp.MustCompile(`^0.*0.*0$`),
			input:  `00000000000000000000x0`,
		},
		{
			name:   `1*1*1`,
			glob:   []string{"1", "", "1", "", "1"},
			regexp: regexp.MustCompile(`^1.*1.*1$`),
			input:  `11111111111111111111x1`,
		},
		{
			name:   `2*2*2*`,
			glob:   []string{"2", "", "2", "", "2", ""},
			regexp: regexp.MustCompile(`^2.*2.*2.*$`),
			input:  `22222222222222222222x2`,
		},
	}

	for _, tt := range tests {
		tt := tt
		b.Run(tt.name, func(b *testing.B) {
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
