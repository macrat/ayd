//go:build linux || darwin
// +build linux darwin

package probe

// osDependsAutoDecode in Unix OS is just an alias of defaultAutoDecode.
// This function only accepts UTF-8 text without BOM.
func osDependsAutoDecode(bytes []byte) string {
	return defaultAutoDecode(bytes)
}
