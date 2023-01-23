package scheme_test

import (
	"net/url"
	"regexp"
	"strings"
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestSFTPScheme_Probe(t *testing.T) {
	t.Parallel()

	server := StartSSHServer(t)

	AssertProbe(t, []ProbeTest{
		{"sftp://pasusr:foobar@" + server.Addr, api.StatusHealthy, strings.Join([]string{
			"directory exists",
			"---",
			"file_count: 2",
			"mtime: 20[-0-9]{8}T[0-9:Z+-]*",
			"permission: 755",
			"type: directory",
		}, "\n"), ""},
		{"sftp://keyusr@" + server.Addr + "/empty?identityfile=" + url.QueryEscape(server.BareKey), api.StatusHealthy, strings.Join([]string{
			"directory exists",
			"---",
			"file_count: 0",
			"mtime: 20[-0-9]{8}T[0-9:Z+-]*",
			"permission: 755",
			"type: directory",
		}, "\n"), ""},
		{"sftp://pasusr:foobar@" + server.Addr + "/hello/world?fingerprint=" + url.QueryEscape(server.FingerprintSHA), api.StatusHealthy, strings.Join([]string{
			"file exists",
			"---",
			"file_size: 11",
			"mtime: 20[-0-9]{8}T[0-9:Z+-]*",
			"permission: 644",
			"type: file",
		}, "\n"), ""},
		{"sftp://pasusr:foobar@" + server.Addr + "/no-such-file", api.StatusFailure, "no such file or directory", ""},

		{"sftp://" + server.Addr, api.StatusUnknown, "", "username is required"},
		{"sftp://pasusr:incorrect@" + server.Addr, api.StatusFailure, strings.Join([]string{
			`failed to connect: ssh: handshake failed: ssh: unable to authenticate, attempted methods \[none password\], no supported methods remain`,
			`---`,
			`fingerprint: ` + regexp.QuoteMeta(server.FingerprintSHA),
			`source_addr: (127\.0\.0\.1|\[::1\]):[0-9]+`,
			`target_addr: (127\.0\.0\.1|\[::1\]):[0-9]+`,
		}, "\n"), ""},
		{"sftp://nosftp:nosftp@" + server.Addr, api.StatusFailure, strings.Join([]string{
			`failed to establish SFTP connection: ssh: subsystem request failed`,
			`---`,
			`fingerprint: ` + regexp.QuoteMeta(server.FingerprintSHA),
			`source_addr: (127\.0\.0\.1|\[::1\]):[0-9]+`,
			`target_addr: (127\.0\.0\.1|\[::1\]):[0-9]+`,
		}, "\n"), ""},
		{"sftp://foo:bar@localhost:10", api.StatusFailure, `(127\.0\.0\.1|\[::1\]):10: connection refused`, ""},
	}, 10)
}

func TestSFTPScheme_Alert(t *testing.T) {
	t.Parallel()

	server := StartSSHServer(t)

	AssertAlert(t, []ProbeTest{
		{"sftp://pasusr:foobar@" + server.Addr + "/incidents.log", api.StatusHealthy, "wrote 140 bytes to file", ""},
		{"sftp://pasusr:foobar@" + server.Addr, api.StatusFailure, `failed to open target file: sftp: "invalid argument" \(SSH_FX_FAILURE\)`, ""},
		{"sftp://pasusr:incorrect@" + server.Addr, api.StatusFailure, "failed to connect: ssh: handshake failed: ssh: unable to authenticate, attempted methods \\[none password\\], no supported methods remain\n---\nfingerprint: " + regexp.QuoteMeta(server.FingerprintSHA) + "\nsource_addr: (127\\.0\\.0\\.1|\\[::1\\]):[0-9]+\ntarget_addr: (127\\.0\\.0\\.1|\\[::1\\]):[0-9]+", ""},
		{"sftp://nosftp:nosftp@" + server.Addr, api.StatusFailure, strings.Join([]string{
			`failed to establish SFTP connection: ssh: subsystem request failed`,
			`---`,
			`fingerprint: ` + regexp.QuoteMeta(server.FingerprintSHA),
			`source_addr: (127\.0\.0\.1|\[::1\]):[0-9]+`,
			`target_addr: (127\.0\.0\.1|\[::1\]):[0-9]+`,
		}, "\n"), ""},
	}, 10)

	AssertProbe(t, []ProbeTest{
		{"sftp://pasusr:foobar@" + server.Addr, api.StatusHealthy, strings.Join([]string{
			"directory exists",
			"---",
			"file_count: 3",
			"mtime: 20[-0-9]{8}T[0-9:Z+-]*",
			"permission: 755",
			"type: directory",
		}, "\n"), ""},
		{"sftp://pasusr:foobar@" + server.Addr + "/incidents.log", api.StatusHealthy, strings.Join([]string{
			"file exists",
			"---",
			"file_size: 140",
			"mtime: 20[-0-9]{8}T[0-9:Z+-]*",
			"permission: 644",
			"type: file",
		}, "\n"), ""},
	}, 10)
}
