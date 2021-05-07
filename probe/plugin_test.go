package probe_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
	"github.com/macrat/ayd/testutil"
)

func TestPluginProbe(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current path: %s", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", origPath+string(filepath.ListSeparator)+filepath.Join(cwd, "testdata"))
	t.Cleanup(func() {
		os.Setenv("PATH", origPath)
	})

	AssertProbe(t, []ProbeTest{
		{"plug:", store.STATUS_HEALTHY, "check plug:", ""},
		{"plug:hello-world", store.STATUS_HEALTHY, "check plug:hello-world", ""},
		{"plug:empty", store.STATUS_HEALTHY, "", ""},
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

	t.Run("plug:invalid-record", func(t *testing.T) {
		p, err := probe.New("plug:invalid-record")
		if err != nil {
			t.Errorf("failed to create plugin: %s", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		rs := testutil.RunCheck(ctx, p)

		if len(rs) != 2 {
			t.Fatalf("got unexpected number of results: %d", len(rs))
		}

		if rs[0].Target.String() != "plug:invalid-record" {
			t.Errorf("got a record of unexpected target: %s", rs[0].Target)
		}

		if rs[1].Target.String() != "ayd:probe:plugin:plug:invalid-record" {
			t.Errorf("got a record of unexpected target: %s", rs[1].Target)
		}
		if rs[1].Status != store.STATUS_FAILURE {
			t.Errorf("got unexpected status: %s", rs[1].Status)
		}
		if rs[1].Message != "invalid record: unexpected column count: \"this is invalid\"" {
			t.Errorf("got unexpected message: %s", rs[1].Message)
		}
	})
}
