package scheme

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

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
		{"http", []string{"http"}},
		{"source-view", []string{"source", "source-view"}},
		{"hello-world+abc-def", []string{"hello", "hello-world", "hello-world+abc", "hello-world+abc-def"}},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			output := pluginCandidates(tt.Input)
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
		{"plag-what", "", ErrUnsupportedScheme},
		{"plag", "", ErrUnsupportedScheme},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			output, err := findPlugin(tt.Input, "probe")
			if err != tt.Error {
				t.Errorf("unexpected error: %s", err)
			}
			if output != tt.Output {
				t.Errorf("unexpected output: %q", output)
			}
		})
	}
}
