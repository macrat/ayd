package scheme_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

func TestPluginScheme_Alert(t *testing.T) {
	t.Parallel()
	PreparePluginPath(t)

	AssertAlert(t, []ProbeTest{
		{"foo:hello-world", api.StatusHealthy, "\"foo:hello-world 2001-02-03T16:05:06Z FAILURE 123.456 dummy:failure test-message\"", ""},
	}, 5)
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
