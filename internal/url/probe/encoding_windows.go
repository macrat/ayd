//go:build windows
// +build windows

package probe

import (
	"reflect"
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

const (
	windowsUTF16LE = 1200
	windowsUTF16BE = 1201
	windowsUTF8    = 65001
)

var (
	windowsCodePages = map[uint32]encoding.Encoding{
		windowsUTF16LE: unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
		windowsUTF16BE: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
		windowsUTF8:    unicode.UTF8,

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
		return defaultAutoDecode(bytes)
	}

	enc, ok := windowsCodePages[codepage]
	if !ok {
		return defaultAutoDecode(bytes)
	}

	bs, err := enc.NewDecoder().Bytes(bytes)
	if err != nil {
		return defaultAutoDecode(bytes)
	}

	return string(bs)
}

func useBOM(defaultCodePage uint32, bytes []byte) (codePage uint32, bytesWithoutBOM []byte) {
	if len(bytes) >= 3 && reflect.DeepEqual(bytes[:3], []byte{0xEF, 0xBB, 0xBF}) {
		return windowsUTF8, bytes[3:]
	}
	if len(bytes) >= 2 {
		if reflect.DeepEqual(bytes[:2], []byte{0xFE, 0xFF}) {
			return windowsUTF16BE, bytes[2:]
		}
		if reflect.DeepEqual(bytes[:2], []byte{0xFF, 0xFE}) {
			return windowsUTF16LE, bytes[2:]
		}
	}
	return defaultCodePage, bytes
}

// osDependsAutoDecode in Windows OS decodes unknown encoding text.
// This function tries UTF-8 first, and then tries the default encoding of the current environment.
// It uses UTF-8 or UTF-16 instead of the default encoding if the text includes BOM.
func osDependsAutoDecode(bytes []byte) string {
	return decodeCodePage(useBOM(windows.GetACP(), bytes))
}
