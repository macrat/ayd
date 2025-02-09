//go:build githubci
// +build githubci

package scheme_test

import (
	"os"
)

func init() {
	os.Setenv("AYD_PING_PRIVILEGED", "true")
}
