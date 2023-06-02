package scheme_test

import (
	"context"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestSFTPScheme_Probe(t *testing.T) {
	t.Parallel()

	server := StartSSHServer(t)

	dummyKey, _ := GenerateSSHKey(t)
	dummyPath := SaveSSHKey(t, dummyKey, "dummy_rsa", "")

	AssertProbe(t, []ProbeTest{
		{"sftp://pasusr:foobar@" + server.Addr, api.StatusHealthy, strings.Join([]string{
			"directory exists",
			"---",
			"file_count: 3",
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

		{"sftp://keyusr@" + server.Addr + "/empty?identityfile=" + url.QueryEscape(dummyPath), api.StatusFailure, strings.Join([]string{
			`failed to connect: ssh: handshake failed: ssh: unable to authenticate, attempted methods \[none publickey\], no supported methods remain`,
			`---`,
			`fingerprint: ` + regexp.QuoteMeta(server.FingerprintSHA),
			`source_addr: (127\.0\.0\.1|\[::1\]):[0-9]+`,
			`target_addr: (127\.0\.0\.1|\[::1\]):[0-9]+`,
		}, "\n"), ""},
	}, 10)

	AssertTimeout(t, "sftp://pasusr:foobar@"+server.Addr)

	t.Run("key-removed", func(t *testing.T) {
		p := testutil.NewProber(t, "sftp://keyusr@"+server.Addr+"/empty?identityfile="+url.QueryEscape(dummyPath))

		os.Remove(dummyPath)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		rs := testutil.RunProbe(ctx, p)

		if len(rs) != 1 {
			t.Fatalf("unexpected number of records:\n%v", rs)
		}

		if rs[0].Message != "no such identity file: "+dummyPath {
			t.Errorf("unexpected message: %q", rs[0].Message)
		}

		if rs[0].Status != api.StatusUnknown {
			t.Errorf("unexpected status: %s", rs[0].Status)
		}
	})
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
			"file_count: 4",
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
