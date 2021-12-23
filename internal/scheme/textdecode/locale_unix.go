//go:build linux || darwin
// +build linux darwin

package textdecode

import (
	"golang.org/x/text/encoding/unicode"
)

// localeDecoder in Unix OS always returns UTF8 decoder.
func localeDecoder() decoder {
	return unicode.UTF8.NewDecoder()
}
