package query

import (
	"strings"
)

type stringMatcher interface {
	Match(string) bool
}

type globMatcher struct {
	Prefix       string
	Suffix       string
	Chunks       []string
	ChunksLength int
}

func (g globMatcher) Match(s string) bool {
	if len(s) < len(g.Prefix)+g.ChunksLength+len(g.Suffix) {
		return false
	}

	if s[:len(g.Prefix)] != g.Prefix || s[len(s)-len(g.Suffix):] != g.Suffix {
		return false
	}

	left := len(g.Prefix)
	right := len(s) - g.ChunksLength - len(g.Suffix)

	for _, chunk := range g.Chunks {
		right += len(chunk)
		i := strings.Index(s[left:right], chunk)
		if i == -1 {
			return false
		}
		left += i + len(chunk)
	}

	return true
}

type exactMatcher struct {
	Str string
}

func (e exactMatcher) Match(s string) bool {
	return e.Str == s
}

type prefixMatcher struct {
	Str string
}

func (p prefixMatcher) Match(s string) bool {
	return strings.HasPrefix(s, p.Str)
}

type suffixMatcher struct {
	Str string
}

func (s suffixMatcher) Match(str string) bool {
	return strings.HasSuffix(str, s.Str)
}

type globBuilder struct {
	prefixClosed bool
	prefix       string
	noSuffix     bool
	chunks       []string
	chunksLength int
	esc          bool
}

func (b *globBuilder) Reset() {
	b.prefixClosed = false
	b.prefix = ""
	b.noSuffix = false
	b.chunks = nil
	b.chunksLength = 0
	b.esc = false
}

func (b *globBuilder) FeedLiteral(s string) {
	b.noSuffix = false
	if b.prefixClosed {
		b.chunks = append(b.chunks, s)
		b.chunksLength += len(s)
	} else {
		b.prefix = s
		b.prefixClosed = true
	}
}

func (b *globBuilder) FeedStar() {
	b.noSuffix = true
	b.prefixClosed = true
}

// Build a matcher from the current state of the builder.
func (b *globBuilder) Build() stringMatcher {
	if !b.prefixClosed {
		return exactMatcher{}
	}

	var suffix string
	if len(b.chunks) > 0 && !b.noSuffix {
		suffix = b.chunks[len(b.chunks)-1]
		b.chunks = b.chunks[:len(b.chunks)-1]
		b.chunksLength -= len(suffix)
	}

	if len(b.chunks) == 0 {
		if b.prefix != "" && suffix == "" {
			if b.noSuffix {
				return prefixMatcher{Str: b.prefix}
			} else {
				return exactMatcher{Str: b.prefix}
			}
		}
		if b.prefix == "" && suffix != "" {
			return suffixMatcher{Str: suffix}
		}
	}

	return globMatcher{
		Prefix:       b.prefix,
		Suffix:       suffix,
		Chunks:       b.chunks,
		ChunksLength: b.chunksLength,
	}
}

// makeGlob makes a new stringMatcher from a list of strings.
// A string in the list means a literal string, and nil means a wildcard.
// For example, "hello*world" in glob syntax is represented as ["hello", nil, "world"].
func makeGlob(query []*string) stringMatcher {
	if len(query) == 0 {
		return exactMatcher{}
	}

	var glob globBuilder
	var buf strings.Builder
	for _, s := range query {
		if s == nil {
			if buf.Len() > 0 {
				glob.FeedLiteral(buf.String())
				buf.Reset()
			}
			glob.FeedStar()
		} else {
			buf.WriteString(*s)
		}
	}
	if buf.Len() > 0 {
		glob.FeedLiteral(buf.String())
	}
	return glob.Build()
}
