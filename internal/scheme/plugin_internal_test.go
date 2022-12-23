package scheme

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func SetCurrentTime(t *testing.T, ct time.Time) {
	currentTime = func() time.Time {
		return ct
	}
	t.Cleanup(func() {
		currentTime = time.Now
	})
}

// PreparePluginPath set PATH to ./testdata/ directory.
func PreparePluginPath(t *testing.T) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current path: %s", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", origPath+string(filepath.ListSeparator)+filepath.Join(cwd, "testdata"))

	// There is not cleanup mechanism because testing.T.Cleanup is sometimes being unstable for parallel test.
}

func TestPluginCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Input  string
		Output []string
	}{
		{"http", []string{"ayd-http-scheme", "ayd-http-probe"}},
		{"source-view", []string{"ayd-source-scheme", "ayd-source-probe", "ayd-source-view-scheme", "ayd-source-view-probe"}},
		{"hello-world+abc-def", []string{"ayd-hello-scheme", "ayd-hello-probe", "ayd-hello-world-scheme", "ayd-hello-world-probe", "ayd-hello-world+abc-scheme", "ayd-hello-world+abc-probe", "ayd-hello-world+abc-def-scheme", "ayd-hello-world+abc-def-probe"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Input, func(t *testing.T) {
			output := pluginCandidates(tt.Input, "probe")
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
		Scope  string
		Input  string
		Output string
		Error  error
	}{
		{"probe", "plug-plus", "ayd-plug-plus-probe", nil},
		{"probe", "plug-minus", "ayd-plug-probe", nil},
		{"probe", "plug", "ayd-plug-probe", nil},
		{"probe", "plag-what", "", ErrUnsupportedScheme},
		{"probe", "plag", "", ErrUnsupportedScheme},
		{"alert", "plug", "ayd-plug-scheme", nil},
		{"alert", "foo", "ayd-foo-alert", nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Input, func(t *testing.T) {
			output, err := findPlugin(tt.Input, tt.Scope)
			if err != tt.Error {
				t.Errorf("unexpected error: %s", err)
			}
			if output != tt.Output {
				t.Errorf("unexpected output: %q", output)
			}
		})
	}
}
