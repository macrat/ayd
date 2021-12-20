//go:build linux || darwin
// +build linux darwin

package textdecode

import (
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// localeDecoder in Unix OS always returns UTF8 decoder.
func localeDecoder() transform.Transformer {
	return unicode.UTF8.NewDecoder()
}
