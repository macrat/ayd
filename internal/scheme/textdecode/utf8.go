package textdecode

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/unicode"
)

// utf8Override is a decoder to try to decode as UTF8.
// If the input is invalid as a UTF8 text, it will uses Fallback.
type utf8Override struct {
	Fallback decoder
}

func (u utf8Override) Bytes(b []byte) ([]byte, error) {
	if utf8.Valid(b) {
		return unicode.UTF8.NewDecoder().Bytes(b)
	} else {
		return u.Fallback.Bytes(b)
	}
}

// UTF8 decodes []byte to string as UTF8.
func UTF8(b []byte) (string, error) {
	b, err := unicode.UTF8.NewDecoder().Bytes(b)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(strings.ReplaceAll(string(b), "\r\n", "\n"), "\r", "\n"), nil
}
