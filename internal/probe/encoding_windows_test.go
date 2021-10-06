//go:build windows
// +build windows

package probe

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestDecodeCodePage(t *testing.T) {
	tests := []struct {
		CodePage uint32
		File     string
		Expect   string
	}{
		{932, "./testdata/utf8", "こんにちはWôrÏd\n"},
		{1252, "./testdata/utf8", "こんにちはWôrÏd\n"},
		{932, "./testdata/cp932", "こんにちは世界\n"},
		{1252, "./testdata/cp1252", "HèÍÎö¨WôrÏd\n"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("cp%d:%s", tt.CodePage, tt.File), func(t *testing.T) {
			f, err := os.Open(tt.File)
			if err != nil {
				t.Fatalf("failed to open test file: %s", err)
			}

			input, err := io.ReadAll(f)
			if err != nil {
				t.Fatalf("failed to read test file: %s", err)
			}

			output := decodeCodePage(tt.CodePage, input)

			if output != tt.Expect {
				t.Fatalf("expected %#v but got %#v", tt.Expect, output)
			}
		})
	}
}

func TestDetectBOM(t *testing.T) {
	tests := []struct {
		Bytes    uint32
		CodePage uint32
	}{
		{[]byte("\xEF\xBB\xBFhello"), 65001}, // UTF-8
		{[]byte("\xFE\xFFhello"), 1200},      // UTF-16LE
		{[]byte("\xFF\xFEhello"), 1201},      // UTF-16BE
		{[]byte("hello"), 0},                 // US-ASCII
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.CodePage), func(t *testing.T) {
			cp := detectBOM(tt.Bytes, 0)

			if cp != tt.CodePage {
				t.Errorf("expected %d but got %d", tt.CodePage, cp)
			}
		})
	}
}
