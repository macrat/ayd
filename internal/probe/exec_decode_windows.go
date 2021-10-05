//go:build windows
// +build windows

package probe

import (
	"unicode/utf8"

	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
)

var (
	windowsCodePages = map[uint32]encoding.Encoding{
		1200:  unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
		1201:  unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
		65001: unicode.UTF8,

		1250: charmap.Windows1250,
		1251: charmap.Windows1251,
		1252: charmap.Windows1252,
		1253: charmap.Windows1253,
		1254: charmap.Windows1254,
		1255: charmap.Windows1255,
		1256: charmap.Windows1256,
		1257: charmap.Windows1257,
		1258: charmap.Windows1258,

		932:   japanese.ShiftJIS,
		20932: japanese.EUCJP,
		50220: japanese.ISO2022JP,
		50221: japanese.ISO2022JP,
		50222: japanese.ISO2022JP,

		949: korean.EUCKR,
		936: simplifiedchinese.GBK,

		950:   traditionalchinese.Big5,
		54936: simplifiedchinese.GB18030,
	}
)

func decodeCodePage(codepage uint32, bytes []byte) string {
	if utf8.Valid(bytes) {
		return string(bytes)
	}

	enc, ok := windowsCodePages[codepage]
	if !ok {
		return string(bytes)
	}

	bs, err := enc.NewDecoder().Bytes(bytes)
	if err != nil {
		return string(bytes)
	}

	return string(bs)
}

func autoDecode(bytes []byte) string {
	return decodeCodePage(windows.GetACP(), bytes)
}
