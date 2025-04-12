package scheme_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestExecLocalScheme_Probe(t *testing.T) {
	if runtime.GOOS != "windows" {
		// This test in windows sometimes be fail if enable parallel.
		// Maybe it's because of the timing to unset path to testdata/dos_polyfill.
		t.Parallel()
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current path: %s", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", origPath+string(filepath.ListSeparator)+filepath.Join(cwd, "testdata", "dos_polyfill"))
	t.Cleanup(func() {
		os.Setenv("PATH", origPath)
	})

	cwd = filepath.ToSlash(cwd)

	AssertProbe(t, []ProbeTest{
		{"exec:./testdata/test?message=hello&code=0", api.StatusHealthy, "hello\n---\nexit_code: 0", ""},
		{"exec:./testdata/test?message=world&code=1", api.StatusFailure, "world\n---\nexit_code: 1", ""},
		{"exec:./testdata/test?message=::foo::bar&code=1", api.StatusFailure, "---\nexit_code: 1\nfoo: bar", ""},
		{"exec:" + path.Join(cwd, "testdata/test") + "?message=hello&code=0", api.StatusHealthy, "hello\n---\nexit_code: 0", ""},
		{"exec:sleep#10", api.StatusFailure, `probe timed out`, ""},
		{"exec:echo#::status::unknown", api.StatusUnknown, "---\nexit_code: 0", ""},
		{"exec:echo#::status::failure", api.StatusFailure, "---\nexit_code: 0", ""},
	}, 5)

	AssertTimeout(t, "exec:echo")
}

func TestExecLocalScheme_Probe_unknownError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.bat")

	f, err := os.Create(file)
	if err != nil {
		t.Fatalf("failed to create test file: %s", err)
	}
	if err := f.Chmod(0766); err != nil {
		t.Fatalf("failed to change permission of test file: %s", err)
	}
	f.Close()

	p := testutil.NewProber(t, "exec:"+file)

	if err := os.Chmod(file, 0000); err != nil {
		t.Fatalf("failed to change permission of test file: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		rs := testutil.RunProbe(ctx, p)
		if rs[0].Status != api.StatusUnknown || !strings.Contains(rs[0].Message, "permission denied") {
			t.Errorf("unexpected result:\n%s", rs[0])
		}
	}

	if err := os.Remove(file); err != nil {
		t.Fatalf("failed to remove test file: %s", err)
	}

	rs := testutil.RunProbe(ctx, p)
	if rs[0].Status != api.StatusUnknown || (!strings.Contains(rs[0].Message, "no such file or directory") && !strings.Contains(rs[0].Message, "file does not exist")) {
		t.Errorf("unexpected result:\n%s", rs[0])
	}
}

func BenchmarkExecLocalScheme(b *testing.B) {
	p := testutil.NewProber(b, "exec:echo#hello-world")

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Probe(ctx, r)
	}
}

func TestExecSSHScheme_Probe(t *testing.T) {
	t.Parallel()

	server := StartSSHServer(t)

	extra := fmt.Sprintf("fingerprint: %s\nsource_addr: [^ ]+\ntarget_addr: %s", regexp.QuoteMeta(server.FingerprintSHA), server.Addr)

	AssertProbe(t, []ProbeTest{
		{"exec+ssh://pasusr:foobar@localhost:10/cmd", api.StatusUnknown, `failed to connect: (\[::1\]|127\.0\.0\.1):10: connection refused`, ""},
		{"exec+ssh://pasusr:foobar@" + server.Addr + "/cmd", api.StatusHealthy, "exec \"/cmd\"\n---\nexit_code: 0\n" + extra, ""},
		{"exec+ssh://pasusr:foobar@" + server.Addr + "/hello#world", api.StatusHealthy, "exec \"/hello\" \"world\"\n---\nexit_code: 0\n" + extra, ""},
		{"exec+ssh://pasusr:foobar@" + server.Addr + "/this%20is#a%20test", api.StatusHealthy, "exec \"/this is\" \"a test\"\n---\nexit_code: 0\n" + extra, ""},
		{"exec+ssh://pasusr:foobar@" + server.Addr + "/cmd?fingerprint=" + url.QueryEscape(server.FingerprintSHA), api.StatusHealthy, "exec \"/cmd\"\n---\nexit_code: 0\n" + extra, ""},
		{"exec+ssh://pasusr:foobar@" + server.Addr + "/cmd?fingerprint=" + url.QueryEscape(server.FingerprintMD5), api.StatusHealthy, "exec \"/cmd\"\n---\nexit_code: 0\n" + extra, ""},
		{"exec+ssh://pasusr:foobar@" + server.Addr + "/cmd?fingerprint=SHA256%3Aaaaaaaaaaa", api.StatusUnknown, "failed to connect: ssh: handshake failed: fingerprint unmatched\n---\n" + extra, ""},
		{"exec+ssh://pasusr:foobar@" + server.Addr + "/error", api.StatusFailure, "exec \"/error\"\n---\nexit_code: 1\n" + extra, ""},
		{"exec+ssh://keyusr@" + server.Addr + "/this/is/key?identityfile=" + url.QueryEscape(server.BareKey), api.StatusHealthy, "exec \"/this/is/key\"\n---\nexit_code: 0\n" + extra, ""},
		{"exec+ssh://keyusr@" + server.Addr + "/env?foo=bar&identityfile=" + url.QueryEscape(server.BareKey), api.StatusHealthy, "env foo=bar\nexec \"/env\"\n---\nexit_code: 0\n" + extra, ""},
		{"exec+ssh://keyusr@" + server.Addr + "/not-found?identityfile=" + url.QueryEscape(server.BareKey), api.StatusUnknown, "exec \"/not-found\"\n---\nexit_code: 127\n" + extra, ""},

		{"exec+ssh://pasusr@" + server.Addr + "/cmd", api.StatusUnknown, "", "password or identityfile is required"},
		{"exec+ssh://pasusr:foobar@" + server.Addr, api.StatusUnknown, "", "missing command"},
		{"exec+ssh://pasusr:abc@" + server.Addr + "/cmd", api.StatusUnknown, "failed to connect: ssh: handshake failed: ssh: unable to authenticate, attempted methods \\[none password\\], no supported methods remain\n---\n" + extra, ""},
	}, 10)

	AssertTimeout(t, "ssh://pasusr:foobar@"+server.Addr+"/cmd")

	t.Run("interrupt", func(t *testing.T) {
		p := testutil.NewProber(t, "exec+ssh://pasusr:foobar@"+server.Addr+"/slow")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		rs := testutil.RunProbe(ctx, p)

		if len(rs) != 1 {
			t.Fatalf("unexpected number for records found:\n%#v", rs)
		}
		if rs[0].Status != api.StatusFailure {
			t.Errorf("expected FAILURE but got %s", rs[0].Status)
		}
		if rs[0].Message != "probe timed out" {
			t.Errorf("unexpected message: %q", rs[0].Message)
		}
		if rs[0].Latency > 50*time.Millisecond {
			t.Errorf("probe took too long: %s", rs[0].Latency)
		}
	})
}

func TestExecSSHScheme_Alert(t *testing.T) {
	t.Parallel()

	server := StartSSHServer(t)

	a, err := scheme.NewAlerter("exec+ssh://keyusr:helloworld@" + server.Addr + "/alert.sh?identityfile=" + server.EncryptedKey)
	if err != nil {
		t.Fatalf("failed to prepare ExecSSHScheme: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r := &testutil.DummyReporter{}

	a.Alert(ctx, r, api.Record{
		Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
		Status:  api.StatusFailure,
		Latency: 123456 * time.Microsecond,
		Target:  &api.URL{Scheme: "dummy", Fragment: "hello"},
		Message: "hello world",
		Extra: map[string]interface{}{
			"hello": "world",
		},
	})

	if len(r.Records) != 1 {
		t.Errorf("unexpected number of records\n%v", r.Records)
	}

	expectedMessage := []string{
		`exec "/alert.sh"`,
		`env ayd_extra={"hello":"world"}`,
		`env ayd_latency=123.456`,
		`env ayd_message=hello world`,
		`env ayd_status=FAILURE`,
		`env ayd_target=dummy:#hello`,
		`env ayd_time=2021-01-02T15:04:05Z`,
	}
	sort.Strings(expectedMessage)

	actualMessage := strings.Split(r.Records[0].Message, "\n")
	sort.Strings(actualMessage)

	if diff := cmp.Diff(expectedMessage, actualMessage); diff != "" {
		t.Errorf("unexpected message:\n%s", diff)
	}

	if len(r.Records[0].Extra) != 4 {
		t.Errorf("unexpected number of extra:\n%#v", r.Records[0].Extra)
	}

	if r.Records[0].Extra["exit_code"] != 0 {
		t.Errorf("unexpected exit_code: expected 0 but got %v", r.Records[0].Extra["exit_code"])
	}

	if r.Records[0].Extra["fingerprint"] != server.FingerprintSHA {
		t.Errorf("unexpected fingerprint: expected %v but got %v", server.FingerprintSHA, r.Records[0].Extra["fingerprint"])
	}

	if addr, ok := r.Records[0].Extra["source_addr"]; !ok || addr == "" {
		t.Errorf("source_addr hast not provided: %v", r.Records[0].Extra["source_addr"])
	}

	if r.Records[0].Extra["target_addr"] != server.Addr {
		t.Errorf("unexpected target_addr: expected %v but got %v", server.Addr, r.Records[0].Extra["target_addr"])
	}
}
