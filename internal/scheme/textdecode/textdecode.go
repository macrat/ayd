package textdecode

import (
	"strings"
)

// decoder is an interface for text decoding.
type decoder interface {
	Bytes(b []byte) ([]byte, error)
}

// Bytes decodes []byte to string.
func Bytes(b []byte) (string, error) {
	dec := localeDecoder()
	b, dec = bomOverride(b, dec)
	s, err := dec.Bytes(b)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(strings.ReplaceAll(string(s), "\r\n", "\n"), "\r", "\n"), nil
}
