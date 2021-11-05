package scheme_test

import (
	"testing"

	"github.com/macrat/ayd/internal/scheme"
)

func TestSplitScheme(t *testing.T) {
	tests := []struct {
		Input     string
		SubScheme string
		Separator rune
		Variant   string
	}{
		{"http", "http", 0, ""},
		{"http-get", "http", '-', "get"},
		{"source+exec", "source", '+', "exec"},
		{"plug-ab+cd", "plug", '-', "ab+cd"},
		{"plug+ab-cd", "plug", '+', "ab-cd"},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			subScheme, separator, variant := scheme.SplitScheme(tt.Input)

			if subScheme != tt.SubScheme {
				t.Errorf("expected sub scheme %#v but got %#v", tt.SubScheme, subScheme)
			}

			if separator != tt.Separator {
				t.Errorf("expected separator %#v but got %#v", tt.Separator, separator)
			}

			if variant != tt.Variant {
				t.Errorf("expected variant %#v but got %#v", tt.Variant, variant)
			}
		})
	}
}
