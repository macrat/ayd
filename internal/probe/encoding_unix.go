//go:build linux || darwin
// +build linux darwin

package probe

func osDependsAutoDecode(bytes []byte) string {
	return string(bytes)
}
