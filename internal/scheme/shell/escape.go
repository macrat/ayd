package shell

import (
	"strings"
)

func Escape(s string) string {
	characters := []byte("\\$`\"")

	for _, c := range characters {
		cs := string([]byte{c})
		s = strings.ReplaceAll(s, cs, `\`+cs)
	}
	return `"` + s + `"`
}
