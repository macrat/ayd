package textdecode

import (
	"unicode/utf8"

	"golang.org/x/text/transform"
	"golang.org/x/text/encoding/unicode"
)

// tryUTF8 is a Transformer to try to decode as UTF8.
// If the input is invalid as a UTF8 text, it will uses Fallback.
type tryUTF8 struct {
	trans transform.Transformer

	Fallback transform.Transformer
}

func (d *tryUTF8) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	if d.trans == nil {
		if utf8.Valid(src) {
			d.trans = unicode.UTF8.NewDecoder()
		} else {
			d.trans = d.Fallback
		}
	}

	return d.trans.Transform(dst, src, atEOF)
}

func (d *tryUTF8) Reset() {
	d.trans = nil
}
