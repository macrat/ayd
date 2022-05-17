package ayd

import (
	"testing"
)

func TestBarePathInOpaque(t *testing.T) {
	tests := []struct {
		Input  string
		Output bool
	}{
		{"exec:///path/to/file", false},
		{"exec:/path/to/file", true},
		{"dummy:", false},
		{"dummy:healthy", true},
		{"http://example.com/path/to/file", false},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			if barePathInOpaque(tt.Input) != tt.Output {
				t.Errorf("expected %#v but got %#v", tt.Output, !tt.Output)
			}
		})
	}
}
