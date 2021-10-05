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
		{932, "./testdata/utf8", "こんにちはWôrÏd"},
		{1252, "./testdata/utf8", "こんにちはWôrÏd"},
		{932, "./testdata/cp932", "こんにちは世界"},
		{1252, "./testdata/cp1252", "HèÍÎö¨WôrÏd"},
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
