//go:build windows
// +build windows

package probe

import (
	"fmt"
	"io"
	"os"
	"reflect"
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

func TestUseBOM(t *testing.T) {
	tests := []struct {
		Input    []byte
		Output   []byte
		CodePage uint32
	}{
		{[]byte("\xEF\xBB\xBFhello"), []byte("hello"), 65001}, // UTF-8
		{[]byte("\xFF\xFEhello"), []byte("hello"), 1200},      // UTF-16LE
		{[]byte("\xFE\xFFhello"), []byte("hello"), 1201},      // UTF-16BE
		{[]byte("hello"), []byte("hello"), 0},                 // US-ASCII
		{[]byte("\xFEhello"), []byte("\xFEhello"), 0},         // US-ASCII
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.CodePage), func(t *testing.T) {
			cp, output := useBOM(0, tt.Input)

			if cp != tt.CodePage {
				t.Errorf("expected code page is %d but got %d", tt.CodePage, cp)
			}

			if !reflect.DeepEqual(output, tt.Output) {
				t.Errorf("expected output is %#v but got %#v", tt.Output, output)
			}
		})
	}
}

func TestOSDependsAutoDecode(t *testing.T) {
	tests := []struct {
		File   string
		Expect string
	}{
		{"./testdata/utf8", "こんにちはWôrÏd"},
		{"./testdata/utf8bom", "UTF8:BOM付き"},
		{"./testdata/utf16be", "UTF16BE:大端"},
		{"./testdata/utf16le", "UTF16LE:리틀 엔디안"},
	}

	for _, tt := range tests {
		t.Run(tt.File, func(t *testing.T) {
			f, err := os.Open(tt.File)
			if err != nil {
				t.Fatalf("failed to open test file: %s", err)
			}

			input, err := io.ReadAll(f)
			if err != nil {
				t.Fatalf("failed to read test file: %s", err)
			}

			output := osDependsAutoDecode(input)

			if output != tt.Expect {
				t.Fatalf("expected %#v but got %#v", tt.Expect, output)
			}
		})
	}
}
