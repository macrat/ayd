//go:build container
// +build container

package scheme_test

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestExecSSHScheme_withContainer(t *testing.T) {
	t.Parallel()

	ResetTestContainer(t, "ssh")
	defer ResetTestContainer(t, "ssh")

	fingerprint := GetContainerSSHFingerprint(t)
	key := GenerateContainerSSHKey(t)

	extra := func(code int) string {
		return fmt.Sprintf(
			"---\nexit_code: %d\nfingerprint: %s\nsource_addr: (127\\.0\\.0\\.1|\\[::1\\]):[0-9]+\ntarget_addr: (127\\.0\\.0\\.1|\\[::1\\]):22",
			code,
			regexp.QuoteMeta(fingerprint),
		)
	}

	AssertProbe(t, []ProbeTest{
		{"exec+ssh://foo:bar@localhost/bin/ls#/home/foo/.ssh", api.StatusHealthy, "authorized_keys\n" + extra(0), ""},
		{"exec+ssh://foo@localhost/bin/ls?identityfile=" + url.QueryEscape(key) + "#/home/foo/.ssh", api.StatusHealthy, "authorized_keys\n" + extra(0), ""},

		{"exec+ssh://foo:bar@localhost/usr/local/bin/make-file?ayd_test_content=hello+world#/tmp/data/ssh/hello", api.StatusHealthy, extra(0), ""},
		{"exec+ssh://foo:bar@localhost/bin/cat#/tmp/data/ssh/hello", api.StatusHealthy, "hello world\n" + extra(0), ""},

		{"exec+ssh://foo:bar@localhost/usr/local/bin/no-such-command", api.StatusUnknown, "ash: /usr/local/bin/no-such-command: not found\n" + extra(127), ""},
	}, 10)

	env := strings.Join([]string{
		`ayd_extra={"hello":"world"}`,
		`ayd_latency=123\.456`,
		`ayd_message=test-message`,
		`ayd_status=FAILURE`,
		`ayd_target=dummy:failure`,
		`ayd_time=2001-02-03T16:05:06Z`,
	}, "\n")
	AssertAlert(t, []ProbeTest{
		{"exec+ssh://foo:bar@localhost/usr/local/bin/list-ayd-env", api.StatusHealthy, env + "\n" + extra(0), ""},
		{"exec+ssh://foo:bar@localhost/usr/local/bin/list-ayd-env", api.StatusHealthy, env + "\n" + extra(0), ""},
		{"exec+ssh://foo:bar@localhost/usr/local/bin/list-ayd-env", api.StatusHealthy, env + "\n" + extra(0), ""},
	}, 10)
}
