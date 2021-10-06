package probe

import (
	"strings"
)

func replaceNewlineChars(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

func removeBOM(s string) string {
	if len(s) >= 3 && s[:3] == "\xEF\xBB\xBF" {
		return s[3:]
	}
	return s
}

func autoDecode(bytes []byte) string {
	return removeBOM(replaceNewlineChars(osDependsAutoDecode(bytes)))
}
