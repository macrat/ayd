package query

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type tokenType int

const (
	lparenToken tokenType = iota
	rparenToken
	orToken
	notToken
	atomToken
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
	case atomToken:
		return "ATOM"
	}
	panic("unreachable")
}

type token struct {
	Type  tokenType
	Value atomValue
}

type atomValue struct {
	Key   stringMatcher
	Value valueMatcher
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
	if strings.ToLower(t.remain[:len(s)]) != s {
		return false
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
		t.scanAtom()
	}
	return true
}

func (t *tokenizer) tokenizeAtom() (hasLeft bool, left []*string, opCode operator, opStr string, right []*string) {
	var buf strings.Builder
	closeBuf := func() {
		if buf.Len() == 0 {
			return
		}
		s := buf.String()
		if opCode == 0 {
			left = append(left, &s)
		} else {
			right = append(right, &s)
		}
		buf.Reset()
	}

	escape := false
	quote := false
	i := 0

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
		} else if quote {
			buf.WriteByte(t.remain[i])
		} else if len(left) == 0 && len(t.remain) > i+1 && (t.remain[i:i+2] == "!=" || t.remain[i:i+2] == "<>") {
			closeBuf()
			opStr = t.remain[i : i+2]
			opCode = opNotEqual
			i++
			continue
		} else if len(left) == 0 && (c == '<' || c == '>' || c == '=') {
			closeBuf()
			switch c {
			case '<':
				opCode = opLessThan
			case '>':
				opCode = opGreaterThan
			case '=':
				opCode = opEqual
			}
			if len(t.remain) > i+1 && t.remain[i+1] == '=' {
				opStr = t.remain[i : i+2]
				opCode |= opEqual
				i++
			} else {
				opStr = t.remain[i : i+1]
			}
			continue
		} else if c == '*' {
			closeBuf()
			if opCode == 0 {
				left = append(left, nil)
			} else {
				right = append(right, nil)
			}
		} else if c == ' ' || c == '\t' || c == '\n' || c == '(' || c == ')' {
			break
		} else {
			buf.WriteByte(t.remain[i])
		}

		if opCode == 0 {
			hasLeft = true
		}
	}

	closeBuf()

	t.remain = t.remain[i:]

	return
}

func (t *tokenizer) scanAtom() {
	hasLeft, left, opCode, opStr, right := t.tokenizeAtom()

	appendLiteral := func(s string) {
		if len(left) == 0 || left[len(left)-1] == nil {
			left = append(left, &s)
		} else {
			l := *left[len(left)-1] + s
			left[len(left)-1] = &l
		}
	}

	mergeLeftAndRight := func() {
		appendLiteral(opStr)
		if len(right) > 0 && right[0] != nil {
			appendLiteral(*right[0])
			left = append(left, right[1:]...)
		} else {
			left = append(left, right...)
		}
		opCode = 0
		right = nil
	}

	// The right side can not be empty if the operator is not an equality operator.
	// For example, `message=` means the message is empty, but `latency>` makes no sense.
	if len(right) == 0 && opCode & ^(opEqual|opNotEqual) != 0 {
		appendLiteral(opStr)
		opCode = 0
	}

	var value valueMatcher

	// When operator is an order comparison operator, the right side should be numeric, time, or duration.
	// If it's not, the query should be treated as just a normal string.
	if opCode&(opLessThan|opGreaterThan) != 0 {
		if len(right) != 1 {
			// If the right side includes glob star, it will be treated as a string.
			mergeLeftAndRight()
		} else {
			if f, err := strconv.ParseFloat(*right[0], 64); err == nil {
				value = numberValueMatcher{Op: opCode, Value: f}
			} else if d, err := time.ParseDuration(*right[0]); err == nil {
				value = durationValueMatcher{Op: opCode, Value: d}
			} else if t, err := api.ParseTime(*right[0]); err == nil {
				value = timeValueMatcher{Op: opCode, Value: t}
			} else {
				mergeLeftAndRight()
			}
		}
	}

	if value == nil {
		if opCode == 0 {
			value = stringValueMatcher{
				Not:     opCode == opNotEqual,
				Matcher: makeGlob(left),
			}
		} else {
			value = stringValueMatcher{
				Not:     opCode == opNotEqual,
				Matcher: makeGlob(right),
			}
		}
	}

	if !hasLeft {
		// TODO: This can be optimized by checking the type of the right side.
		// For example, if the right side is a duration, the query can only be matched with the latency.
		t.buf = token{Type: atomToken, Value: atomValue{
			Value: value,
		}}
	} else {
		// TODO: The left side must be a field name, even though it seems empty (it can be happened in some specific queries like `""=value`).
		// There should care about the left side is exactMatcher or other stringMatcher, because exactMatcher can be more faster than checking all fields using stringMatcher.
		t.buf = token{Type: atomToken, Value: atomValue{
			Key:   makeGlob(left),
			Value: value,
		}}
	}

	fmt.Printf("hasLeft: %v\n", hasLeft)
	fmt.Printf("left: %#v\n", left)
	fmt.Printf("op: %v(%04b)\n", opStr, opCode)
	fmt.Printf("right: %#v\n", right)
}

func (t *tokenizer) Token() token {
	return t.buf
}
