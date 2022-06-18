package scheme_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestFileScheme_Probe(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current path: %s", err)
	}

	testDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(testDir, "no-permission"), 0); err != nil {
		t.Fatalf("failed to make test directory")
	}

	fileOutput := strings.Join([]string{
		"file exists",
		"---",
		"file_size: 47",
		"mtime: [-+:0-9TZ]+",
		"permission: 755",
		"type: file",
	}, "\n")

	dirOutput := strings.Join([]string{
		"directory exists",
		"---",
		"file_count: 26",
		"mtime: [-+:0-9TZ]+",
		"permission: 755",
		"type: directory",
	}, "\n")

	forbiddenDirOutput := strings.Join([]string{
		"directory exists",
		"---",
		"mtime: [-+:0-9TZ]+",
		"permission: 000",
		"type: directory",
	}, "\n")

	AssertProbe(t, []ProbeTest{
		{"file:./testdata/test", api.StatusHealthy, fileOutput, ""},
		{"file:./testdata/test#this%20is%20file", api.StatusHealthy, fileOutput, ""},
		{"file:" + cwd + "/testdata/test", api.StatusHealthy, fileOutput, ""},
		{"file:testdata", api.StatusHealthy, dirOutput, ""},
		{"file:" + cwd + "/testdata", api.StatusHealthy, dirOutput, ""},
		{"file:testdata/of-course-no-such-file-or-directory", api.StatusFailure, "no such file or directory", ""},
		{"file:" + filepath.ToSlash(testDir) + "/no-permission", api.StatusHealthy, forbiddenDirOutput, ""},
		{"file:" + filepath.ToSlash(testDir) + "/no-permission/foobar", api.StatusFailure, "permission denied", ""},
	}, 2)
}

func TestFileScheme_Alert(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join(t.TempDir(), "alert.log")

	a, err := scheme.NewAlerter("file:" + filepath.ToSlash(logPath))
	if err != nil {
		t.Fatalf("faield to prepare FileScheme: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r := &testutil.DummyReporter{}

	a.Alert(ctx, r, api.Record{
		Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
		Status:  api.StatusFailure,
		Latency: 123456 * time.Microsecond,
		Target:  &api.URL{Scheme: "dummy", Fragment: "hello"},
		Message: "hello world",
	})

	expected := `{"time":"2021-01-02T15:04:05Z", "status":"FAILURE", "latency":123.456, "target":"dummy:#hello", "message":"hello world"}` + "\n"

	if len(r.Records) != 1 {
		t.Errorf("unexpected number of records\n%v", r.Records)
	} else {
		if r.Records[0].Status != api.StatusHealthy {
			t.Errorf("unexpected status: %s", r.Records[0].Status)
		}
		if r.Records[0].Message != fmt.Sprintf("wrote %d bytes to file", len(expected)) {
			t.Errorf("unexpected status: %s", r.Records[0].Message)
		}
	}

	bytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read output file: %s", err)
	}

	actual := strings.ReplaceAll(strings.ReplaceAll(string(bytes), "\r\n", "\n"), "\r", "\n")

	if actual != expected {
		t.Errorf("unexpected output\n=== want =====\n%s\n=== actual =====\n%s", expected, actual)
	}
}
