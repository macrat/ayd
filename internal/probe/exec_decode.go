//go:build !windows
// +build !windows

package probe

func autoDecode(bytes []byte) string {
	return string(bytes)
}
