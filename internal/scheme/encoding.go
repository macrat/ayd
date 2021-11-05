package scheme

import (
	"strings"

	"golang.org/x/text/encoding/unicode"
)

// replaceNewlineChars replaces CRLF in Windows or CR in Mac OS to LF.
func replaceNewlineChars(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

// autoDecode decodes unknown encoding text to string type.
// It calls osDependsAutoDecode and replaceNewlineChars.
func autoDecode(bytes []byte) string {
	return replaceNewlineChars(osDependsAutoDecode(bytes))
}

// defaultAutoDecode decodes UTF-8 text.
// Invalid characters will replaced by \uFFFD that means unknown character.
// Basiclly, it called by osDependsAutoDecode.
func defaultAutoDecode(bytes []byte) string {
	s, err := unicode.UTF8.NewDecoder().Bytes(bytes)
	if err != nil {
		return string(bytes)
	}
	return string(s)
}
