package scheme_test

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestSourceScheme_Probe(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current path: %s", err)
	}

	server := RunDummyHTTPServer()
	defer server.Close()

	tests := []struct {
		Target       string
		Records      map[string]api.Status
		ErrorPattern string
	}{
		{"source:./testdata/healthy-list.txt", map[string]api.Status{
			"dummy:healthy#sub-list":             api.StatusHealthy,
			"dummy:healthy#healthy-list":         api.StatusHealthy,
			"source:./testdata/healthy-list.txt": api.StatusHealthy,
		}, ""},
		{"source:testdata/healthy-list.txt", map[string]api.Status{
			"dummy:healthy#sub-list":           api.StatusHealthy,
			"dummy:healthy#healthy-list":       api.StatusHealthy,
			"source:testdata/healthy-list.txt": api.StatusHealthy,
		}, ""},
		{"source:./testdata/failure-list.txt", map[string]api.Status{
			"dummy:healthy#sub-list":             api.StatusHealthy,
			"dummy:healthy#failure-list":         api.StatusHealthy,
			"dummy:failure":                      api.StatusFailure,
			"source:./testdata/failure-list.txt": api.StatusHealthy,
		}, ""},
		{"source:./testdata/invalid-list.txt", map[string]api.Status{
			"source:./testdata/invalid-list.txt": api.StatusFailure,
		}, "invalid source URL:\n  ::invalid host::\n  no-such-scheme:hello world\n  source:./testdata/no-such-list.txt"},
		{"source:testdata/invalid-list.txt", map[string]api.Status{
			"source:testdata/invalid-list.txt": api.StatusFailure,
		}, "invalid source URL:\n  ::invalid host::\n  no-such-scheme:hello world\n  source:./testdata/no-such-list.txt"},
		{"source:./testdata/include-invalid-list.txt", map[string]api.Status{
			"source:./testdata/include-invalid-list.txt": api.StatusFailure,
		}, "invalid source URL:\n  ::invalid host::\n  no-such-scheme:hello world\n  source:./testdata/no-such-list.txt"},
		{"source:./testdata/no-such-list.txt", map[string]api.Status{
			"source:./testdata/no-such-list.txt": api.StatusFailure,
		}, `invalid source: open \./testdata/no-such-list\.txt: (no such file or directory|The system cannot find the file specified\.)`},
		{"source:" + path.Join(cwd, "testdata/sub-list.txt"), map[string]api.Status{
			"dummy:healthy#sub-list":                            api.StatusHealthy,
			"source:" + path.Join(cwd, "testdata/sub-list.txt"): api.StatusHealthy,
		}, ""},
		{"source:" + path.Join(cwd, "testdata/sub-list.txt"), map[string]api.Status{
			"dummy:healthy#sub-list":                            api.StatusHealthy,
			"source:" + path.Join(cwd, "testdata/sub-list.txt"): api.StatusHealthy,
		}, ""},

		{"source+" + server.URL + "/source", map[string]api.Status{
			"dummy:healthy#1":                  api.StatusHealthy,
			"dummy:healthy#2":                  api.StatusHealthy,
			"source+" + server.URL + "/source": api.StatusHealthy,
		}, ""},
		{"source+" + server.URL + "/source/slow", map[string]api.Status{
			"source+" + server.URL + "/source/slow": api.StatusFailure,
		}, ""},

		{"source+exec:./testdata/listing-script?message=abc#world", map[string]api.Status{
			"dummy:healthy#hello": api.StatusHealthy,
			"dummy:healthy#world": api.StatusHealthy,
			"dummy:healthy#abc":   api.StatusHealthy,
			"source+exec:./testdata/listing-script?message=abc#world": api.StatusHealthy,
		}, ""},
		{"source+exec:" + path.Join(cwd, "testdata/listing-script?message=def#ayd"), map[string]api.Status{
			"dummy:healthy#hello": api.StatusHealthy,
			"dummy:healthy#ayd":   api.StatusHealthy,
			"dummy:healthy#def":   api.StatusHealthy,
			"source+exec:" + filepath.ToSlash(path.Join(cwd, "testdata/listing-script?message=def#ayd")): api.StatusHealthy,
		}, ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Target, func(t *testing.T) {
			p, err := scheme.NewProber(tt.Target)
			if err != nil && tt.ErrorPattern == "" {
				t.Fatalf("failed to create probe: %s", err)
			}
			if tt.ErrorPattern != "" {
				if err == nil {
					t.Fatalf("expected error %v but got nil", tt.ErrorPattern)
				} else if ok, _ := regexp.MatchString("^"+tt.ErrorPattern+"$", err.Error()); !ok {
					t.Fatalf("--- expected error ---\n%s\n--- actual error ---\n%s", tt.ErrorPattern, err)
				}
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			rs := testutil.RunProbe(ctx, p)

			if len(rs) != len(tt.Records) {
				t.Fatalf("unexpected number of records: %d\n%v", len(rs), rs)
			}

			for _, r := range rs {
				expect, ok := tt.Records[r.Target.String()]
				if !ok {
					t.Errorf("got unexpected or duplicated record: %s", r.Target)
					continue
				}
				if r.Status != expect {
					t.Errorf("got unexpected status: %s: expected %s but got %s", r.Target, expect, r.Status)
				}
				delete(tt.Records, r.Target.String())
			}

			for target := range tt.Records {
				t.Errorf("missing record of %s", target)
			}
		})
	}

	AssertTimeout(t, "source:./testdata/healthy-list.txt")
}

func TestSource_stderr(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	copyFile := func(source, dest string, permission fs.FileMode) {
		s, err := os.Open(source)
		if err != nil {
			t.Fatalf("failed to open test file: %s", err)
		}
		defer s.Close()

		d, err := os.Create(dest)
		if err != nil {
			t.Fatalf("failed to create test file: %s", err)
		}
		defer d.Close()

		_, err = io.Copy(d, s)
		if err != nil {
			t.Fatalf("failed to copy file: %s", err)
		}

		if err := d.Chmod(permission); err != nil {
			t.Fatalf("failed to set permission of test file: %s", err)
		}
	}

	copyFile("./testdata/write-stdout", dir+"/list", 0766)
	copyFile("./testdata/write-stdout.bat", dir+"/list.bat", 0766)

	p, err := scheme.NewProber("source+exec:" + dir + "/list")
	if err != nil {
		t.Fatalf("failed to create probe: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rs := testutil.RunProbe(ctx, p)

	if len(rs) != 2 {
		t.Fatalf("unexpected number of records: %d\n%v", len(rs), rs)
	}

	for _, r := range rs {
		if r.Status != api.StatusHealthy {
			t.Fatalf("unexpected status:\n%s", r)
		}
	}

	copyFile("./testdata/write-stderr", dir+"/list", 0766)
	copyFile("./testdata/write-stderr.bat", dir+"/list.bat", 0766)

	rs = testutil.RunProbe(ctx, p)

	if len(rs) != 1 {
		t.Fatalf("unexpected number of records: %d\n%v", len(rs), rs)
	}

	if rs[0].Status != api.StatusFailure {
		t.Fatalf("unexpected status:\n%s", rs[0])
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		copyFile("./testdata/write-stdout", dir+"/list", 0000)

		rs = testutil.RunProbe(ctx, p)

		if len(rs) != 1 {
			t.Fatalf("unexpected number of records: %d\n%v", len(rs), rs)
		}

		if rs[0].Status != api.StatusFailure {
			t.Fatalf("unexpected status:\n%s", rs[0])
		}
	}
}

func TestSourceScheme_inactiveTargetHandling(t *testing.T) {
	t.Parallel()
	dir := filepath.ToSlash(t.TempDir())

	if err := os.WriteFile(dir+"/list.txt", []byte("dummy:#1\ndummy:#2"), 0644); err != nil {
		t.Fatalf("faield to prepare test list file: %s", err)
	}

	sourceURL := "source:" + dir + "/list.txt"
	p, err := scheme.NewProber(sourceURL)
	if err != nil {
		t.Fatalf("failed to prepare probe: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r := &testutil.DummyReporter{}

	p.Probe(ctx, r)
	r.AssertActives(t, sourceURL, "dummy:#1", "dummy:#2")

	if err := os.WriteFile(dir+"/list.txt", []byte("dummy:#1\ndummy:#3"), 0644); err != nil {
		t.Fatalf("faield to update test list file: %s", err)
	}

	p.Probe(ctx, r)
	r.AssertActives(t, sourceURL, "dummy:#1", "dummy:#3")

	if err := os.WriteFile(dir+"/list.txt", []byte("dummy:#2\ndummy:#3"), 0644); err != nil {
		t.Fatalf("faield to update test list file: %s", err)
	}

	p.Probe(ctx, r)
	r.AssertActives(t, sourceURL, "dummy:#2", "dummy:#3")
}

func TestSourceScheme_Alert(t *testing.T) {
	t.Parallel()
	PreparePluginPath(t)
	dir := filepath.ToSlash(t.TempDir())

	if err := os.WriteFile(dir+"/list.txt", []byte("foo:alert-test\ndummy:\n"), 0644); err != nil {
		t.Fatalf("faield to prepare test list file: %s", err)
	}

	a, err := scheme.NewAlerter("source:" + dir + "/list.txt")
	if err != nil {
		t.Fatalf("failed to prepare alerter: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r := &testutil.DummyReporter{}

	a.Alert(ctx, r, api.Record{
		CheckedAt: time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
		Status:    api.StatusFailure,
		Latency:   123456 * time.Microsecond,
		Target:    &url.URL{Scheme: "dummy", Opaque: "hello-world"},
		Message:   "test-message",
	})

	if len(r.Records) != 3 {
		t.Fatalf("unexpected number of records\n%v", r.Records)
	}

	if r.Records[0].Target.String() != "alert:source:"+dir+"/list.txt" {
		t.Errorf("first record should be source but got %s", r.Records[0])
	}

	if r.Records[1].Target.String() == "alert:dummy:" {
		r.Records[1], r.Records[2] = r.Records[2], r.Records[1]
	}

	expected := `"foo:alert-test 2021-01-02T15:04:05Z FAILURE 123.456 dummy:hello-world test-message"`
	if r.Records[1].Message != expected {
		t.Errorf("unexpected message for foo:alert-test\n--- expected --\n%s\n--- actual ---\n%s", expected, r.Records[1].Message)
	}
}

func BenchmarkSourceScheme_loadProbers(b *testing.B) {
	for _, n := range []int{10, 25, 50, 75, 100, 250, 500, 750, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			f, err := os.CreateTemp("", "ayd-test-*-list.txt")
			if err != nil {
				b.Fatalf("failed to create test file: %s", err)
			}
			defer f.Close()
			defer os.Remove(f.Name())

			for i := 0; i < n; i++ {
				fmt.Fprintf(f, "ping:host-%d\n", i)
			}

			target := &url.URL{Scheme: "source", Opaque: f.Name()}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = scheme.NewSourceProbe(target)
			}
		})
	}
}

func BenchmarkSourceScheme(b *testing.B) {
	for _, n := range []int{10, 25, 50, 75, 100, 250, 500, 750, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			f, err := os.CreateTemp("", "ayd-test-*-list.txt")
			if err != nil {
				b.Fatalf("failed to create test file: %s", err)
			}
			defer f.Close()
			defer os.Remove(f.Name())

			for i := 0; i < n; i++ {
				fmt.Fprintf(f, "dummy:healthy?latency=0s#%d\n", i)
			}

			target := &url.URL{Scheme: "source", Opaque: f.Name()}

			p, err := scheme.NewSourceProbe(target)
			if err != nil {
				b.Fatalf("failed to create probe: %s", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			r := &testutil.DummyReporter{}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.Probe(ctx, r)
			}
		})
	}
}
