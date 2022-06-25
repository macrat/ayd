package scheme_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	if err := os.Mkdir(filepath.Join(testDir, "parent"), 0755); err != nil {
		t.Fatalf("failed to make test directory")
	}
	if err := os.Mkdir(filepath.Join(testDir, "parent/child"), 0); err != nil {
		t.Fatalf("failed to make test directory")
	}

	fileOutput := strings.Join([]string{
		"file exists",
		"---",
		"file_size: 13",
		"mtime: [-+:0-9TZ]+",
		"permission: 644",
		"type: file",
	}, "\n")
	if runtime.GOOS == "windows" {
		fileOutput = strings.Replace(fileOutput, "permission: 644", "permission: 666", 1)
	}

	dirOutput := strings.Join([]string{
		"directory exists",
		"---",
		"file_count: 1",
		"mtime: [-+:0-9TZ]+",
		"permission: 755",
		"type: directory",
	}, "\n")
	if runtime.GOOS == "windows" {
		dirOutput = strings.Replace(dirOutput, "permission: 755", "permission: 777", 1)
	}

	forbiddenDirOutput := strings.Join([]string{
		"directory exists",
		"---",
		"mtime: [-+:0-9TZ]+",
		"permission: 000",
		"type: directory",
	}, "\n")

	hiddenDirOutput := "permission denied"

	AssertProbe(t, []ProbeTest{
		{"file:./testdata/file.txt", api.StatusHealthy, fileOutput, ""},
		{"file:./testdata/file.txt#this%20is%20file", api.StatusHealthy, fileOutput, ""},
		{"file:" + filepath.ToSlash(cwd) + "/testdata/file.txt", api.StatusHealthy, fileOutput, ""},
		{"file:testdata/of-course-no-such-file-or-directory", api.StatusFailure, "no such file or directory", ""},
		{"file:" + filepath.ToSlash(testDir) + "/parent", api.StatusHealthy, dirOutput, ""},
	}, 2)

	if runtime.GOOS != "windows" {
		AssertProbe(t, []ProbeTest{
			{"file:" + filepath.ToSlash(testDir) + "/parent/child", api.StatusHealthy, forbiddenDirOutput, ""},
			{"file:" + filepath.ToSlash(testDir) + "/parent/child/foobar", api.StatusFailure, hiddenDirOutput, ""},
		}, 2)
	}
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

	actual := string(bytes)
	if actual != expected {
		t.Errorf("unexpected output\n=== want =====\n%s\n=== actual =====\n%s", expected, actual)
	}
}
