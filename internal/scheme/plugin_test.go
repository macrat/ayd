package scheme_test

import (
	"context"
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

// PreparePluginPath is implemented in plugin_internal_test.go.
// This is a shorthand for that function.
var PreparePluginPath = scheme.PreparePluginPath

func TestPluginScheme_Probe(t *testing.T) {
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
		{"plug:extra", api.StatusHealthy, "with extra\n---\nhello: world", ""},
		{"ayd:test", api.StatusUnknown, "", "unsupported scheme"},
		{"alert:test", api.StatusUnknown, "", "unsupported scheme"},
	}, 5)

	AssertTimeout(t, "plug:")

	if runtime.GOOS != "windows" {
		t.Run("forbidden:", func(t *testing.T) {
			_, err := scheme.NewProber("forbidden:")
			if err != scheme.ErrUnsupportedScheme {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	t.Run("plug:invalid-record", func(t *testing.T) {
		p, err := scheme.NewProber("plug:invalid-record")
		if err != nil {
			t.Fatalf("failed to create plugin: %s", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		rs := testutil.RunProbe(ctx, p)

		if len(rs) != 2 {
			t.Fatalf("got unexpected number of results: %d", len(rs))
		}

		if rs[0].Target.String() != "plug:invalid-record" {
			t.Errorf("got a record of unexpected target: %s", rs[0].Target)
		}

		if rs[1].Target.String() != "plug:invalid-record" {
			t.Errorf("got a record of unexpected target: %s", rs[1].Target)
		}
		if rs[1].Status != api.StatusUnknown {
			t.Errorf("got unexpected status: %s", rs[1].Status)
		}
		if rs[1].Message != "the plugin reported invalid records" {
			t.Errorf("got unexpected message: %s", rs[1].Message)
		}
		if diff := cmp.Diff(map[string]any{"raw_message": "this is invalid"}, rs[1].Extra); diff != "" {
			t.Errorf("got unexpected extra values: %s", diff)
		}
	})

	t.Run("removed-plug:", func(t *testing.T) {
		origPath := os.Getenv("PATH")
		dir := t.TempDir()
		os.Setenv("PATH", origPath+string(filepath.ListSeparator)+dir)

		if err := os.WriteFile(dir+"/ayd-removed-plug-probe", []byte("#!/bin/sh\n"), 0744); err != nil {
			t.Fatalf("failed to prepare dummy plugin for UNIX: %s", err)
		}
		if err := os.WriteFile(dir+"/ayd-removed-plug-probe.bat", []byte("@echo off\n"), 0744); err != nil {
			t.Fatalf("failed to prepare dummy plugin for Windows: %s", err)
		}

		p, err := scheme.NewProber("removed-plug:")
		if err != nil {
			t.Fatalf("failed to create plugin: %s", err)
		}

		if err = os.Remove(dir + "/ayd-removed-plug-probe"); err != nil {
			t.Fatalf("failed to remove dummy plugin for UNIX: %s", err)
		}
		if err = os.Remove(dir + "/ayd-removed-plug-probe.bat"); err != nil {
			t.Fatalf("failed to remove dummy plugin for Windows: %s", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		rs := testutil.RunProbe(ctx, p)

		if len(rs) != 1 {
			t.Fatalf("got unexpected number of results: %d", len(rs))
		}

		if rs[0].Target.String() != "removed-plug:" {
			t.Errorf("got a record of unexpected target: %s", rs[0].Target)
		}
		if rs[0].Status != api.StatusUnknown {
			t.Errorf("got unexpected status: %s", rs[0].Status)
		}
		if rs[0].Message != "probe plugin for removed-plug was not found" {
			t.Errorf("got unexpected message: %s", rs[0].Message)
		}
	})
}

func TestPluginScheme_Probe_timezone(t *testing.T) {
	PreparePluginPath(t)
	t.Setenv("TZ", "UTC")
	scheme.SetCurrentTime(t, time.Date(2001, 2, 3, 16, 10, 0, 0, time.UTC))

	tests := []struct {
		URL  string
		Time time.Time
	}{
		{"plug:+0900", time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC)},
		{"plug-plus:utc", time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC)},
	}

	for _, tt := range tests {
		p, err := scheme.NewProber(tt.URL)
		if err != nil {
			t.Fatalf("%s: failed to create plugin: %s", tt.URL, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		rs := testutil.RunProbe(ctx, p)

		if len(rs) != 1 {
			t.Fatalf("%s: unexpected number of results: %d", tt.URL, len(rs))
		}

		if rs[0].Target.String() != tt.URL {
			t.Errorf("%s: unexpected target: %s", tt.URL, rs[0].Target)
		}

		if !rs[0].Time.Equal(tt.Time) {
			t.Errorf("%s: unexpected time: %s", tt.URL, rs[0].Time)
		}
	}
}

func TestPluginScheme_Probe_trimTime(t *testing.T) {
	PreparePluginPath(t)
	t.Setenv("TZ", "UTC")

	p, err := scheme.NewProber("plug:")
	if err != nil {
		t.Fatalf("failed to create plugin: %s", err)
	}

	base := time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC)

	tests := []struct {
		Cur time.Time
		Out time.Time
	}{
		{base, base},
		{base.Add(1 * time.Minute), base},
		{base.Add(50 * time.Minute), base},
		{base.Add(60 * time.Minute), base},
		{base.Add(61 * time.Minute), base.Add(1 * time.Minute)},
		{base.Add(-1 * time.Minute), base.Add(-1 * time.Minute)},
	}

	for i, tt := range tests {
		scheme.SetCurrentTime(t, tt.Cur)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		rs := testutil.RunProbe(ctx, p)

		if len(rs) != 1 {
			t.Fatalf("%d: %s: unexpected number of results: %d", i, tt.Cur, len(rs))
		}

		if rs[0].Target.String() != "plug:" {
			t.Errorf("%d: %s: unexpected target: %s", i, tt.Cur, rs[0].Target)
		}

		if !rs[0].Time.Equal(tt.Out) {
			t.Errorf("%d: %s: unexpected time: %s", i, tt.Cur, rs[0].Time)
		}
	}
}

func TestPluginScheme_Alert(t *testing.T) {
	t.Parallel()
	PreparePluginPath(t)

	if runtime.GOOS == "windows" {
		AssertAlert(t, []ProbeTest{
			{"foo:hello-world", api.StatusHealthy, "foo:hello-world\n---\n" + `record: {"time":"2001-02-03T16:05:06Z", "status":"FAILURE", "latency":123.456, "target":"dummy:failure", "message":"test-message", "hello":"world"}`, ""},
		}, 5)
	} else {
		AssertAlert(t, []ProbeTest{
			{"foo:hello-world", api.StatusHealthy, "foo:hello-world\n---\n" + `record: {"hello":"world","latency":123.456,"message":"test-message","status":"FAILURE","target":"dummy:failure","time":"2001-02-03T16:05:06Z"}`, ""},
		}, 5)
	}
}

func TestPluginProbe_inactiveTargetHandling(t *testing.T) {
	t.Parallel()
	PreparePluginPath(t)

	sourceURL := "plug:change"
	p, err := scheme.NewProber(sourceURL)
	if err != nil {
		t.Fatalf("failed to prepare probe: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r := &testutil.DummyReporter{}
	r.Actives = []*api.URL{{Scheme: "plug", Opaque: "change"}}

	p.Probe(ctx, r)
	r.AssertActives(t, "changed:plug")
}
