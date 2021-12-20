package textdecode

import (
	"testing"
	"fmt"

	"golang.org/x/text/transform"
	"golang.org/x/text/encoding"
)

func TestTryUTF8(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{"こんにちはUTF8", "こんにちはUTF8"},
		{"invalid \xFF\xFF", "\uFFFD"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%#v", string(tt.Input)), func(t *testing.T) {
			trans := &tryUTF8{Fallback: encoding.Replacement.NewDecoder()}

			output, _, err := transform.String(trans, tt.Input)
			if err != nil {
				t.Fatalf("failed to transform: %s", err)
			}

			if string(output) != tt.Output {
				t.Errorf("expected %q but got %q", tt.Output, string(output))
			}
		})
	}
}
