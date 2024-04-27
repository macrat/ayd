package query

import (
	"strings"
)

type stringMatcher interface {
	Match(s string) bool
}

type globTokenType int8

type globMatcher struct {
	Tokens []string
	Suffix string
	MinLength int
}

func globMatch(tokens []string, minLength int, suffix, target string) bool {
	switch len(tokens) {
	case 0:
		return true
	case 1:
		return strings.HasPrefix(target, tokens[0])
	}

	if !strings.HasPrefix(target, tokens[0]) {
		return false
	}

	target = target[len(tokens[0]):]
	minLength -= len(tokens[0])

	for i := 0; i < len(target)-minLength; i++ {
		if globMatch(tokens[1:], minLength, suffix, target[i:]) {
			return true
		}
	}

	return false
}

func (g globMatcher) Match(s string) bool {
	if len(g.Tokens) == 0 {
		return s == g.Suffix
	}
	if len(s) < g.MinLength || !strings.HasSuffix(s, g.Suffix) {
		return false
	}
	return globMatch(g.Tokens, g.MinLength, g.Suffix, s)
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
	tokens []string
	buf    strings.Builder
	minLength int
	feeded bool
	withPrefix bool
	withSuffix bool
	esc    bool
}

func newGlobBuilder() *globBuilder {
	return &globBuilder{
		withPrefix: true,
	}
}

func (b *globBuilder) closeBuf() {
	if b.buf.Len() > 0 {
		b.tokens = append(b.tokens, b.buf.String())
		b.minLength += b.buf.Len()
		b.buf.Reset()
	}
}

func (b *globBuilder) Feed(r rune) {
	b.withSuffix = true
	if b.esc {
		switch r {
		case 'n':
			b.buf.WriteRune('\n')
		case 'r':
			b.buf.WriteRune('\r')
		case 't':
			b.buf.WriteRune('\t')
		default:
			b.buf.WriteRune(r)
		}
		b.esc = false
		return
	}

	switch r {
	case '\\':
		b.esc = true
	case '*':
		b.withSuffix = false
		if b.feeded {
			b.closeBuf()
		} else {
			b.withPrefix = false
			b.tokens = append(b.tokens, "")
		}
	default:
		b.buf.WriteRune(r)
	}

	b.feeded = true
}

func (b *globBuilder) Build() stringMatcher {
	b.closeBuf()
	if len(b.tokens) == 0 {
		return globMatcher{}
	}

	if len(b.tokens) == 1 {
		switch {
		case b.withPrefix && b.withSuffix:
			return exactMatcher{b.tokens[0]}
		case b.withPrefix:
			return prefixMatcher{b.tokens[0]}
		}
	}

	if len(b.tokens) == 2 && b.tokens[0] == "" && b.withSuffix {
		return suffixMatcher{b.tokens[1]}
	}


	g := globMatcher{
		Tokens: b.tokens,
		MinLength: b.minLength,
		Suffix: "",
	}

	if b.withSuffix {
		g.Tokens = g.Tokens[:len(g.Tokens)-1]
		g.Suffix = b.tokens[len(b.tokens)-1]
	}

	return g
}
