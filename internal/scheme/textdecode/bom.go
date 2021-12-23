package textdecode

import (
	"golang.org/x/text/encoding/unicode"
)

// bomOverride checks the text has the BOM, and returns decoder for the encoding.
// If the text has no BOM, this functions returns defaultDecoder.
// The []byte in returns is the text that droped BOM.
func bomOverride(b []byte, defaultDecoder decoder) ([]byte, decoder) {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return b[3:], unicode.UTF8.NewDecoder()
	}
	if len(b) >= 2 {
		if b[0] == 0xFE && b[1] == 0xFF {
			return b[2:], unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
		}
		if b[0] == 0xFF && b[1] == 0xFE {
			return b[2:], unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		}
	}
	return b, defaultDecoder
}
