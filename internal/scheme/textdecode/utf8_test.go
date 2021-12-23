package textdecode

import (
	"fmt"
	"testing"

	"golang.org/x/text/encoding"
)

func TestUTF8Override(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{"こんにちはUTF8", "こんにちはUTF8"},
		{"invalid \xFF\xFF", "\uFFFD"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%#v", string(tt.Input)), func(t *testing.T) {
			dec := utf8Override{encoding.Replacement.NewDecoder()}

			output, err := dec.Bytes([]byte(tt.Input))
			if err != nil {
				t.Fatalf("failed to transform: %s", err)
			}

			if string(output) != tt.Output {
				t.Errorf("expected %q but got %q", tt.Output, string(output))
			}
		})
	}
}
