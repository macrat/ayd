package textdecode

import (
	"fmt"
	"testing"
)

func TestNormalizeNewline(t *testing.T) {
	nn := newlineNormalizer{}

	tests := []struct {
		Input  string
		AtEOF  bool
		Output string
		NSrc   int
		NDst   int
	}{
		{"hello\nworld\n\x00\x00", true, "hello\nworld\n", 12, 12},
		{"hello\r\nworld\r\n\x00\x00", true, "hello\nworld\n", 14, 12},
		{"hello\r\nworld\r\n\r", true, "hello\nworld\n\n", 15, 13},
		{"hello\rworld\r", true, "hello\nworld\n", 12, 12},
		{"hello\rworld\r", false, "hello\nworld", 11, 11},
		{"hello\rworld", true, "hello\nworld", 11, 11},
		{"hello\r\n\r\n\nworld\r\n\x00\x00\x00", false, "hello\n\n\nworld\n", 17, 14},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%#v-%v", tt.Input, tt.AtEOF), func(t *testing.T) {
			var dst [1024]byte
			nDst, nSrc, err := nn.Transform(dst[:], []byte(tt.Input), tt.AtEOF)
			if err != nil {
				t.Fatalf("failed to transform: %s", err)
			}

			if nSrc != tt.NSrc {
				t.Errorf("unexpected length of source: expected %d but got %d", tt.NSrc, nSrc)
			}

			if nDst != tt.NDst {
				t.Errorf("unexpected length of destination: expected %d but got %d", tt.NDst, nDst)
			}

			if output := string(dst[:nDst]); output != tt.Output {
				t.Errorf("expected %#v but got %#v", tt.Output, output)
			}
		})
	}
}
