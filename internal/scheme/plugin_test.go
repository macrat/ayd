package scheme_test

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/scheme"
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
}

func TestPluginCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Input  string
		Output []string
	}{
		{"http", []string{"http"}},
		{"source-view", []string{"source", "source-view"}},
		{"hello-world+abc-def", []string{"hello", "hello-world", "hello-world+abc", "hello-world+abc-def"}},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			output := scheme.PluginCandidates(tt.Input)
			if diff := cmp.Diff(output, tt.Output); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestFindPlugin(t *testing.T) {
	t.Parallel()
	PreparePluginPath(t)

	tests := []struct {
		Input  string
		Output string
		Error  error
	}{
		{"plug-plus", "ayd-plug-plus-probe", nil},
		{"plug-minus", "ayd-plug-probe", nil},
		{"plug", "ayd-plug-probe", nil},
		{"plag-what", "", scheme.ErrUnsupportedScheme},
		{"plag", "", scheme.ErrUnsupportedScheme},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			output, err := scheme.FindPlugin(tt.Input, "probe")
			if err != tt.Error {
				t.Errorf("unexpected error: %s", err)
			}
			if output != tt.Output {
				t.Errorf("unexpected output: %q", output)
			}
		})
	}
}

func TestExecutePlugin(t *testing.T) {
	t.Parallel()
	PreparePluginPath(t)

	tests := []struct {
		Target  *url.URL
		Status  api.Status
		Message string
	}{
		{&url.URL{Scheme: "plug"}, api.StatusHealthy, "check plug:"},
		{&url.URL{Scheme: "plug-plus"}, api.StatusHealthy, "plus plugin: plug-plus:"},
		{&url.URL{Scheme: "no-such"}, api.StatusUnknown, "probe plugin for no-such was not found"},
	}

	for _, tt := range tests {
		t.Run(tt.Target.String(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			r := &testutil.DummyReporter{}
			scheme.ExecutePlugin(ctx, r, "probe", tt.Target, []string{tt.Target.String()}, nil)

			r.Lock()

			if len(r.Records) != 1 {
				t.Fatalf("unexpected length of records\n%v", r.Records)
			}

			if r.Records[0].Status != tt.Status {
				t.Errorf("unexpected status: %s\n", r.Records[0].Status)
			}

			if r.Records[0].Message != tt.Message {
				t.Errorf("unexpected message: %s\n", r.Records[0].Message)
			}
		})
	}
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
		{"plug-plus:hello", api.StatusHealthy, "plus plugin: plug-plus:hello", ""},
		{"plug:empty", api.StatusHealthy, "", ""},
		{"ayd:test", api.StatusUnknown, "", "unsupported scheme"},
		{"alert:test", api.StatusUnknown, "", "unsupported scheme"},
	}, 5)

	AssertTimeout(t, "plug:")

	if runtime.GOOS != "windows" {
		t.Run("forbidden:", func(t *testing.T) {
			_, err := scheme.NewProbe("forbidden:")
			if err != scheme.ErrUnsupportedScheme {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	t.Run("plug:invalid-record", func(t *testing.T) {
		p, err := scheme.NewProbe("plug:invalid-record")
		if err != nil {
			t.Fatalf("failed to create plugin: %s", err)
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

func TestWithoutPluginProbe(t *testing.T) {
	PreparePluginPath(t)

	tests := []struct {
		URL                string
		NewError           error
		WithoutPluginError error
	}{
		{"dummy:healthy", nil, nil},
		{"plug:test", nil, scheme.ErrUnsupportedScheme},
		{"::", scheme.ErrInvalidURL, scheme.ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.URL, func(t *testing.T) {
			p, err := scheme.NewProbe(tt.URL)
			if err != tt.NewError {
				t.Fatalf("scheme.NewProbe: unexpected error: %s", err)
			}

			_, err = scheme.WithoutPluginProbe(p, err)
			if err != tt.WithoutPluginError {
				t.Fatalf("scheme.WithoutPluginProbe: unexpected error: %s", err)
			}
		})
	}
}

func TestPluginProbe_inactiveTargetHandling(t *testing.T) {
	t.Parallel()
	PreparePluginPath(t)

	sourceURL := "plug:change"
	p, err := scheme.NewProbe(sourceURL)
	if err != nil {
		t.Fatalf("failed to prepare probe: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r := &testutil.DummyReporter{}
	r.Actives = []*url.URL{{Scheme: "plug", Opaque: "change"}}

	p.Check(ctx, r)
	r.AssertActives(t, "changed:plug")
}
