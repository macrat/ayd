package textdecode

import (
	"bytes"

	"golang.org/x/text/transform"
)

type newlineNormalizer struct{}

func (nn newlineNormalizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	if len(dst) < len(src) {
		return 0, 0, transform.ErrShortDst
	}
	copy(dst, src)

	nSrc = bytes.IndexByte(src, byte(0))
	end := nSrc
	if nSrc < 0 {
		nSrc = len(src)
		end = nSrc - 1
	}

	initial := bytes.IndexByte(src, byte('\r'))
	if initial < 0 {
		nDst = nSrc
		return
	}

	shift := 0
	for i := initial; i+shift < end; i++ {
		if dst[i+shift] == byte('\r') {
			if dst[i+shift+1] == byte('\n') {
				shift++
			} else {
				dst[i+shift] = byte('\n')
			}
		}
		dst[i] = dst[i+shift]
	}

	if src[len(src)-1] == byte('\r') {
		if atEOF {
			dst[nSrc-shift-1] = byte('\n')
		} else {
			nSrc--
		}
	}

	nDst = nSrc - shift

	return
}

func (nn newlineNormalizer) Reset() {}
