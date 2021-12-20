package textdecode_test

import (
	"testing"
	"os"
	"strings"

	"github.com/macrat/ayd/internal/scheme/textdecode"
)

func Test_characterHandling(t *testing.T) {
	tests := []struct{
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
			output, err := textdecode.ToString(strings.NewReader(tt.Input))
			if err != nil {
				t.Errorf("ToString: failed to decode %#v: %s", tt.Input, err)
			} else if output != tt.Output {
				t.Errorf("ToString: expected %#v but got %#v", tt.Output, output)
			}
		})
	}
}

func Test_unicode(t *testing.T) {
	tests := []struct {
		File     string
		Expect   string
	}{
		{"./testdata/utf8", "こんにちはWôrÏd"},
		{"./testdata/utf8bom", "UTF8:BOM付き"},
		{"./testdata/utf16be", "UTF16BE:大端"},
		{"./testdata/utf16le", "UTF16LE:리틀 엔디안"},
	}

	for _, tt := range tests {
		t.Run(tt.File, func(t *testing.T) {
			output, err := DecodeFile(tt.File)
			if err != nil {
				t.Errorf("failed to decode: %s", err)
			} else if output != tt.Expect {
				t.Errorf("expected %#v but got %#v", tt.Expect, output)
			}
		})
	}
}

func DecodeFile(fname string) (string, error) {
	f, err := os.Open(fname)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return textdecode.ToString(f)
}
