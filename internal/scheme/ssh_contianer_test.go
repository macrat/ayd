//go:build container
// +build container

package scheme_test

import (
	"context"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
	"golang.org/x/crypto/ssh"
)

func RunSSHCommandOnContainer(t testing.TB, command, arg string) string {
	u, err := api.ParseURL("exec+ssh://foo:bar@localhost:2222" + command + "#" + url.PathEscape(arg))
	if err != nil {
		t.Fatalf("failed to parse URL for prepare test: %s", err)
	}
	s, err := scheme.NewExecSSHScheme(u)
	if err != nil {
		t.Fatalf("failed to make scheme for prepare test: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rs := testutil.RunProbe(ctx, s)

	if len(rs) != 1 {
		t.Fatalf("unexpected number of records during prepare to test: %s", err)
	}

	return rs[0].Message
}

func GetContainerSSHFingerprint(t testing.TB) string {
	return RunSSHCommandOnContainer(t, "/bin/cat", "fingerprint")
}

func GenerateContainerSSHKey(t testing.TB) (keyPath string) {
	pri, pub := GenerateSSHKey(t)

	authorizedKey := string(ssh.MarshalAuthorizedKey(pub))

	RunSSHCommandOnContainer(t, "/usr/local/bin/store-sshkey", authorizedKey)

	return SaveSSHKey(t, pri, "id_rsa", "")
}

func TestSSHProbe_Probe_withContainer(t *testing.T) {
	t.Parallel()

	fingerprint := GetContainerSSHFingerprint(t)
	key := GenerateContainerSSHKey(t)

	extra := "---\nfingerprint: " + regexp.QuoteMeta(fingerprint) + "\nsource_addr: (127\\.0\\.0\\.1|\\[::1\\]):[0-9]+\ntarget_addr: (127\\.0\\.0\\.1|\\[::1\\]):2222"

	AssertProbe(t, []ProbeTest{
		{"ssh://foo:bar@localhost:2222", api.StatusHealthy, "succeed to connect\n" + extra, ""},
		{"ssh://foo:bar@localhost:2222?fingerprint=" + url.QueryEscape(fingerprint), api.StatusHealthy, "succeed to connect\n" + extra, ""},
		{"ssh://foo@localhost:2222?fingerprint=" + url.QueryEscape(fingerprint) + "&identityfile=" + url.QueryEscape(key), api.StatusHealthy, "succeed to connect\n" + extra, ""},
	}, 10)
}
