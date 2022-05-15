package ayd

import (
	"net/url"
)

func isFragmentCodePoint(c byte) bool {
	return (c == 0x21 ||
		c == 0x24 ||
		(0x26 <= c && c <= 0x3b) ||
		c == 0x3d ||
		(0x3f <= c && c <= 0x5a) ||
		c == 0x5f ||
		(0x61 <= c && c <= 0x7a) ||
		c == 0x7e ||
		0x80 <= c)
}

const hex = "0123456789ABCDEF"

func escapeFragment(s string) string {
	var buf [1024]byte
	var ss []byte

	if len(s)*3 <= len(buf) {
		ss = buf[:len(s)*3]
	} else {
		ss = make([]byte, len(s)*3)
	}

	j := 0
	for i := 0; i < len(s); i++ {
		if c := s[i]; isFragmentCodePoint(c) {
			ss[j] = c
			j++
		} else {
			ss[j] = '%'
			ss[j+1] = hex[c>>4]
			ss[j+2] = hex[c&15]
			j += 3
		}
	}

	return string(ss[:j])
}

// URL is a target URL.
type URL url.URL

// ParseURL parses string as a URL.
func ParseURL(s string) (*URL, error) {
	u, err := url.Parse(s)
	return (*URL)(u), err
}

// ToURL converts Ayd URL to *url.URL in standard library.
func (u *URL) ToURL() *url.URL {
	return (*url.URL)(u)
}

// String returns string version of the URL.
// The password in the URL will be masked.
func (u *URL) String() string {
	s := u.ToURL().Redacted()
	if u.Fragment != "" {
		l := len(u.ToURL().EscapedFragment())
		s = s[:len(s)-l] + escapeFragment(u.Fragment)
	}
	return s
}
