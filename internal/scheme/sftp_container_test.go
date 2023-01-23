//go:build container
// +build container

package scheme_test

import (
	"net/url"
	"strings"
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestSFTPScheme_withContainer(t *testing.T) {
	t.Parallel()

	ResetTestContainer(t, "ssh")
	defer ResetTestContainer(t, "ssh")

	fingerprint := GetContainerSSHFingerprint(t)
	key := GenerateContainerSSHKey(t)

	AssertProbe(t, []ProbeTest{
		{"sftp://foo:bar@localhost:2222/tmp/data/ssh?fingerprint=" + url.QueryEscape(fingerprint), api.StatusHealthy, strings.Join([]string{
			"directory exists",
			"---",
			"file_count: 0",
			"mtime: 20[-0-9]{8}T[0-9:Z+-]*",
			"permission: 777",
			"type: directory",
		}, "\n"), ""},
		{"sftp://foo@localhost:2222/tmp/data/ssh?identityfile=" + url.QueryEscape(key), api.StatusHealthy, strings.Join([]string{
			"directory exists",
			"---",
			"file_count: 0",
			"mtime: 20[-0-9]{8}T[0-9:Z+-]*",
			"permission: 777",
			"type: directory",
		}, "\n"), ""},
		{"sftp://foo:bar@localhost:2222/tmp/data", api.StatusHealthy, strings.Join([]string{
			"directory exists",
			"---",
			"file_count: 2",
			"mtime: 20[-0-9]{8}T[0-9:Z+-]*",
			"permission: 755",
			"type: directory",
		}, "\n"), ""},
		{"sftp://foo:bar@localhost:2222/tmp/data/ssh/no-such-file", api.StatusFailure, "no such file or directory", ""},
	}, 10)

	AssertAlert(t, []ProbeTest{
		{"sftp://foo:bar@localhost:2222/tmp/data/ssh/incidents.log", api.StatusHealthy, "wrote 140 bytes to file", ""},
	}, 10)

	AssertProbe(t, []ProbeTest{
		{"sftp://foo:bar@localhost:2222/tmp/data/ssh", api.StatusHealthy, strings.Join([]string{
			"directory exists",
			"---",
			"file_count: 1",
			"mtime: 20[-0-9]{8}T[0-9:Z+-]*",
			"permission: 777",
			"type: directory",
		}, "\n"), ""},
		{"sftp://foo:bar@localhost:2222/tmp/data/ssh/incidents.log", api.StatusHealthy, strings.Join([]string{
			"file exists",
			"---",
			"file_size: 140",
			"mtime: 20[-0-9]{8}T[0-9:Z+-]*",
			"permission: 644",
			"type: file",
		}, "\n"), ""},
	}, 10)

	AssertAlert(t, []ProbeTest{
		{"sftp://foo:bar@localhost:2222/tmp/data/ssh/incidents.log", api.StatusHealthy, "wrote 140 bytes to file", ""},
	}, 10)

	AssertProbe(t, []ProbeTest{
		{"sftp://foo:bar@localhost:2222/tmp/data/ssh/incidents.log", api.StatusHealthy, strings.Join([]string{
			"file exists",
			"---",
			"file_size: 280",
			"mtime: 20[-0-9]{8}T[0-9:Z+-]*",
			"permission: 644",
			"type: file",
		}, "\n"), ""},
	}, 10)
}
