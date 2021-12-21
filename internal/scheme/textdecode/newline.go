package textdecode

import (
	"bytes"

	"golang.org/x/text/transform"
)

type newlineNormalizer struct{}

func (nn newlineNormalizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	nSrc = bytes.IndexByte(src, byte(0))
	if nSrc < 0 {
		nSrc = len(src)
	}
	if nSrc == 0 {
		return
	}

	x := src[:nSrc]
	x = bytes.ReplaceAll(x, []byte("\r\n"), []byte("\n"))
	x = bytes.ReplaceAll(x, []byte("\r"), []byte("\n"))

	if len(x) > len(dst) {
		return 0, 0, transform.ErrShortDst
	}

	copy(dst, x)
	nDst = len(x)

	if !atEOF && src[nSrc-1] == byte('\r') {
		nSrc--
		nDst--
	}

	return
}

func (nn newlineNormalizer) Reset() {}
