//go:build linux || darwin
// +build linux darwin

package textdecode

import (
	"golang.org/x/text/encoding/unicode"
)

// localeDecoder in Unix OS is an UTF8 decoder.
var localeDecoder decoder = unicode.UTF8.NewDecoder()
