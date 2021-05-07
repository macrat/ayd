package ayd

import (
	"testing"
)

func TestUnescapeMessage(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{`hello world`, `hello world`},
		{`"hello"world`, `"hello"world`},
		{`hello\tworld`, "hello\tworld"},
		{`hello\nworld`, "hello\nworld"},
		{`hello\r\nworld`, "hello\\r\nworld"},
		{`\\hello\\world\\\\\n`, "\\hello\\world\\\\\n"},
		{`\n`, "\n"},
		{``, ""},
	}

	for _, tt := range tests {
		got := unescapeMessage(tt.Input)
		if got != tt.Output {
			t.Errorf("%#v: unexpected result\nexpected: %#v\n but got: %#v", tt.Input, tt.Output, got)
		}
	}
}

func TestEscapeMessage(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{`hello world`, `hello world`},
		{`"hello"world`, `"hello"world`},
		{"hello\tworld", `hello\tworld`},
		{"\thello\tworld\t", `\thello\tworld\t`},
		{"\n\nhello\nworld\n", `\n\nhello\nworld\n`},
		{`\n\t\\`, `\\n\\t\\\\`},
		{"\n", `\n`},
		{"", ``},
	}

	for _, tt := range tests {
		got := escapeMessage(tt.Input)
		if got != tt.Output {
			t.Errorf("%#v: unexpected result\nexpected: %#v\n but got: %#v", tt.Input, tt.Output, got)
		}
	}
}
