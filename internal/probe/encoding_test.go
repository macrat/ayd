package probe

import (
	"testing"
)

func TestAutoDecode(t *testing.T) {
	tests := []struct {
		Name   string
		Input  string
		Output string
	}{
		{"CRLF", "hello\r\n\r\nworld\r\n", "hello\n\nworld\n"},
		{"CR", "hello\r\rworld\r", "hello\n\nworld\n"},
		{"LF", "hello\n\nworld\n", "hello\n\nworld\n"},
		{"mixed", "hello\n\r\r\nworld\r\n", "hello\n\n\nworld\n"},
		{"invalid-character", "hello\xFF\xFFworld", "hello\uFFFD\uFFFDworld"},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output := autoDecode([]byte(tt.Input))
			if output != tt.Output {
				t.Errorf("expected %#v but got %#v", tt.Output, output)
			}
		})
	}
}
