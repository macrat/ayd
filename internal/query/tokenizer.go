package query

import (
	"fmt"
	"strings"
)

type tokenType int

const (
	lparenToken tokenType = iota
	rparenToken
	orToken
	notToken
	simpleKeywordToken
	fieldKeywordToken
)

func (t tokenType) String() string {
	switch t {
	case lparenToken:
		return "LPAREN"
	case rparenToken:
		return "RPAREN"
	case orToken:
		return "OR"
	case notToken:
		return "NOT"
	case simpleKeywordToken:
		return "SIMPLE_KEYWORD"
	case fieldKeywordToken:
		return "FIELD_KEYWORD"
	}
	panic("unreachable")
}

type token struct {
	Type  tokenType
	Key   stringMatcher
	Not   bool
	Value valueMatcher
}

func (t token) String() string {
	if t.Value != nil {
		not := ""
		if t.Not {
			not = "!"
		}
		if t.Key != nil {
			return fmt.Sprintf("%s%s(%v %#v)", not, t.Type, t.Key, t.Value)
		} else {
			return fmt.Sprintf("%s%s(%#v)", not, t.Type, t.Value)
		}
	}
	return t.Type.String()
}

type tokenizer struct {
	remain string
	buf    token
}

func newTokenizer(input string) *tokenizer {
	return &tokenizer{remain: input}
}

func (t *tokenizer) check(s string) bool {
	if len(t.remain) < len(s)+1 {
		return false
	}
	for i := range s {
		if toLowerByte(t.remain[i]) != s[i] {
			return false
		}
	}
	return t.remain[len(s)] == ' ' || t.remain[len(s)] == '(' || t.remain[len(s)] == ')'
}

func toLowerByte(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + 'a' - 'A'
	}
	return b
}

func (t *tokenizer) Scan() bool {
	if len(t.remain) == 0 {
		return false
	}

	for t.remain[0] == ' ' || t.remain[0] == '\t' || t.remain[0] == '\n' {
		t.remain = t.remain[1:]
		if len(t.remain) == 0 {
			return false
		}
	}

	c := toLowerByte(t.remain[0])

	switch {
	case c == '(':
		t.buf = token{Type: lparenToken}
		t.remain = t.remain[1:]
	case c == ')':
		t.buf = token{Type: rparenToken}
		t.remain = t.remain[1:]
	case c == 'a' && t.check("and"):
		t.remain = t.remain[3:]
		t.Scan()
	case c == 'o' && t.check("or"):
		t.buf = token{Type: orToken}
		t.remain = t.remain[2:]
	case c == '-' || c == '!':
		t.buf = token{Type: notToken}
		t.remain = t.remain[1:]
	case c == 'n' && t.check("not"):
		t.buf = token{Type: notToken}
		t.remain = t.remain[3:]
	default:
		t.scanKeyword()
	}
	return true
}

type keywordTokenType int

const (
	literalToken keywordTokenType = iota
	operatorToken
)

type keywordToken struct {
	Type  keywordTokenType
	Op    operator
	Value []*string
}

func (t *tokenizer) scanKeyword() {
	tokens := make([]keywordToken, 0, 8)

	var buf strings.Builder
	var strings []*string
	escape := false
	quote := false
	hasOp := false
	startWithQuote := false
	i := 0

	closeBuf := func() {
		if buf.Len() > 0 {
			s := buf.String()
			buf.Reset()
			strings = append(strings, &s)
		}
		if len(strings) > 0 {
			tokens = append(tokens, keywordToken{
				Type:  literalToken,
				Value: strings,
			})
			strings = nil
		}
	}

	for ; i < len(t.remain); i++ {
		c := t.remain[i]

		if escape {
			switch c {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case 'r':
				buf.WriteByte('\r')
			default:
				buf.WriteByte(c)
			}
			escape = false
		} else if c == '\\' {
			escape = true
		} else if c == '"' {
			quote = !quote
			if i == 0 {
				startWithQuote = true
			}
		} else if quote {
			buf.WriteByte(t.remain[i])
		} else if len(t.remain) > i+1 && (t.remain[i:i+2] == "!=" || t.remain[i:i+2] == "<>") {
			closeBuf()

			hasOp = true
			opStr := t.remain[i : i+2]
			opCode := opNotEqual
			tokens = append(tokens, keywordToken{
				Type:  operatorToken,
				Op:    opCode,
				Value: []*string{&opStr},
			})

			i++
			continue
		} else if c == '<' || c == '>' || c == '=' {
			closeBuf()

			hasOp = true
			opStr := t.remain[i : i+1]
			var opCode operator
			switch c {
			case '<':
				opCode = opLessThan
			case '>':
				opCode = opGreaterThan
			case '=':
				opCode = opEqual
			}
			if len(t.remain) > i+1 && t.remain[i+1] == '=' {
				opCode |= opEqual
				opStr += "="
				i++
			}
			tokens = append(tokens, keywordToken{
				Type:  operatorToken,
				Op:    opCode,
				Value: []*string{&opStr},
			})

			continue
		} else if c == '*' {
			closeBuf()
			strings = append(strings, nil)
		} else if c == ' ' || c == '\t' || c == '\n' || c == '(' || c == ')' {
			break
		} else {
			buf.WriteByte(c)
		}
	}

	closeBuf()

	if len(tokens) == 0 {
		t.remain = t.remain[i:]
		t.buf = token{
			Type:  simpleKeywordToken,
			Value: anyValueMatcher{},
		}
		return
	}

	var left []*string
	var right valueMatcher
	var op operator

	if len(tokens) > 1 {
		for i := 0; i < len(tokens); i++ {
			if tokens[i].Type == operatorToken && (tokens[i].Op == opEqual || tokens[i].Op == opNotEqual) {
				op = tokens[i].Op

				for j := 0; j < i; j++ {
					left = append(left, tokens[j].Value...)
				}

				var r []*string
				for j := i + 1; j < len(tokens); j++ {
					r = append(r, tokens[j].Value...)
				}
				right = newValueMatcher(r, op&opNotMask)

				break
			}
		}
	}

	if op == 0 {
		if !hasOp || tokens[len(tokens)-1].Type == operatorToken {
			var r []*string
			for i := 0; i < len(tokens); i++ {
				r = append(r, tokens[i].Value...)
			}
			right = newValueMatcher(r, op&opNotMask)
		} else {
			op = tokens[len(tokens)-2].Op

			var ss []*string
			for i := 0; i < len(tokens)-2; i++ {
				ss = append(ss, tokens[i].Value...)
			}

			var err error
			right, err = newOrderingValueMatcher(tokens[len(tokens)-1].Value, op&opNotMask)
			if err == nil {
				left = ss
			} else {
				ss = append(append(ss, tokens[len(tokens)-2].Value...), tokens[len(tokens)-1].Value...)
				op = opIncludes
				right = newValueMatcher(ss, opIncludes)
			}
		}
	}

	if len(left) != 0 || op != opIncludes && startWithQuote {
		t.buf = token{
			Type:  fieldKeywordToken,
			Key:   newStringMatcher(left),
			Not:   op == opNotEqual,
			Value: right,
		}
	} else {
		t.buf = token{
			Type:  simpleKeywordToken,
			Not:   op == opNotEqual,
			Value: right,
		}
	}

	t.remain = t.remain[i:]

	return
}

func (t *tokenizer) Token() token {
	return t.buf
}
