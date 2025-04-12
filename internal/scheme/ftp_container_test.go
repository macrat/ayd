//go:build container
// +build container

package scheme_test

import (
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestFTPScheme_withContainer(t *testing.T) {
	t.Parallel()

	ResetTestContainer(t, "ftp")
	defer ResetTestContainer(t, "ftp")

	AssertProbe(t, []ProbeTest{
		{"ftp://foo:bar@localhost/", api.StatusHealthy, "directory exists\n---\nfile_count: 0\nmtime: [^ ]+\ntype: directory", ""},
		{"ftp://foo:bar@localhost:21/", api.StatusHealthy, "directory exists\n---\nfile_count: 0\nmtime: [^ ]+\ntype: directory", ""},
		{"ftp://foo:bar@localhost/no-such-file", api.StatusFailure, "no such file or directory", ""},
		{"ftp://foo:bar@localhost/ayd-incidents.log", api.StatusFailure, "no such file or directory", ""},
	}, 10)

	AssertAlert(t, []ProbeTest{
		{"ftp://foo:bar@localhost/ayd-incidents.log", api.StatusHealthy, "uploaded 140 bytes to the server", ""},
	}, 10)

	AssertProbe(t, []ProbeTest{
		{"ftp://foo:bar@localhost/", api.StatusHealthy, "directory exists\n---\nfile_count: 1\nmtime: [^ ]+\ntype: directory", ""},
		{"ftp://foo:bar@localhost/no-such-file", api.StatusFailure, "no such file or directory", ""},
		{"ftp://foo:bar@localhost/ayd-incidents.log", api.StatusHealthy, "file exists\n---\nfile_size: 140\nmtime: [^ ]+\ntype: file", ""},
	}, 10)

	AssertAlert(t, []ProbeTest{
		{"ftp://foo:bar@localhost/ayd-incidents.log", api.StatusHealthy, "uploaded 140 bytes to the server", ""},
	}, 10)

	AssertProbe(t, []ProbeTest{
		{"ftp://foo:bar@localhost/ayd-incidents.log", api.StatusHealthy, "file exists\n---\nfile_size: 280\nmtime: [^ ]+\ntype: file", ""},
	}, 10)
}
