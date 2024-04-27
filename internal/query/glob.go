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

	minlen := g.ChunksLength + len(g.Suffix)

	idx := len(g.Prefix)
	for _, chunk := range g.Chunks {
		minlen -= len(chunk)
		i := strings.Index(s[idx:len(s)-minlen], chunk)
		if i == -1 {
			return false
		}
		idx += i + len(chunk)
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
	buf          strings.Builder
	chunksLength int
	esc          bool
}

func (b *globBuilder) closeBuf() {
	if b.buf.Len() > 0 {
		if b.prefixClosed {
			b.chunks = append(b.chunks, b.buf.String())
			b.chunksLength += b.buf.Len()
		} else {
			b.prefix = b.buf.String()
			b.prefixClosed = true
		}
		b.buf.Reset()
	}
}

func (b *globBuilder) Feed(r rune) {
	b.noSuffix = false

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
		b.noSuffix = true
		if b.buf.Len() == 0 && !b.prefixClosed {
			b.prefixClosed = true
		} else {
			b.closeBuf()
		}
	default:
		b.buf.WriteRune(r)
	}
}

// Build a matcher from the current state of the builder.
// If the interesting flag is false, the matcher will always return true.
func (b *globBuilder) Build() (matcher stringMatcher, interesting bool) {
	if len(b.chunks) == 0 {
		switch {
		case b.prefix != "" && b.noSuffix:
			return prefixMatcher{b.prefix}, true
		case b.prefix == "" && !b.noSuffix:
			if b.prefixClosed {
				return suffixMatcher{b.buf.String()}, true
			} else {
				return exactMatcher{b.buf.String()}, true
			}
		case b.prefix == "" && b.noSuffix:
			return globMatcher{}, false
		}
	}

	return globMatcher{
		Prefix:       b.prefix,
		Suffix:       b.buf.String(),
		Chunks:       b.chunks,
		ChunksLength: b.chunksLength,
	}, true
}
