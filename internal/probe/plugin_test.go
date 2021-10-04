package probe_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/probe"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func PreparePluginPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current path: %s", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", origPath+string(filepath.ListSeparator)+filepath.Join(cwd, "testdata"))
	t.Cleanup(func() {
		os.Setenv("PATH", origPath)
	})
}

func TestPluginProbe(t *testing.T) {
	t.Parallel()
	PreparePluginPath(t)

	AssertProbe(t, []ProbeTest{
		{"plug:", api.StatusHealthy, "check plug:", ""},
		{"plug:hello-world", api.StatusHealthy, "check plug:hello-world", ""},
		{"plug-hello:world", api.StatusHealthy, "check plug-hello:world", ""},
		{"plug+hello:world", api.StatusHealthy, `check plug\+hello:world`, ""},
		{"plug-hello+world:", api.StatusHealthy, `check plug-hello\+world:`, ""},
		{"plug:empty", api.StatusHealthy, "", ""},
		{"ayd:test", api.StatusUnknown, "", "unsupported scheme"},
		{"alert:test", api.StatusUnknown, "", "unsupported scheme"},
	}, 5)

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
		if rs[1].Status != api.StatusUnknown {
			t.Errorf("got unexpected status: %s", rs[1].Status)
		}
		if rs[1].Message != "invalid record: unexpected column count: \"this is invalid\"" {
			t.Errorf("got unexpected message: %s", rs[1].Message)
		}
	})
}

func TestWithoutPlugin(t *testing.T) {
	PreparePluginPath(t)

	tests := []struct {
		URL                string
		NewError           error
		WithoutPluginError error
	}{
		{"dummy:healthy", nil, nil},
		{"plug:test", nil, probe.ErrUnsupportedScheme},
		{"::", probe.ErrInvalidURL, probe.ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.URL, func(t *testing.T) {
			p, err := probe.New(tt.URL)
			if err != tt.NewError {
				t.Fatalf("probe.New: unexpected error: %s", err)
			}

			_, err = probe.WithoutPlugin(p, err)
			if err != tt.WithoutPluginError {
				t.Fatalf("probe.WithoutPlugin: unexpected error: %s", err)
			}
		})
	}
}
