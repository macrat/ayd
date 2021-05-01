package probe_test

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

func TestPluginProbe(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current path: %s", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", origPath+":"+path.Join(cwd, "testdata"))
	t.Cleanup(func() {
		os.Setenv("PATH", origPath)
	})

	AssertProbe(t, []ProbeTest{
		{"plug:", store.STATUS_HEALTHY, "http://localhost:9000 -> plug:"},
		{"plug:hello-world", store.STATUS_HEALTHY, "http://localhost:9000 -> plug:hello-world"},
	})

	AssertTimeout(t, "plug:")

	if runtime.GOOS != "windows" {
		t.Run("forbidden:", func(t *testing.T) {
			_, err := probe.New("forbidden:")
			if err != probe.ErrUnsupportedScheme {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
