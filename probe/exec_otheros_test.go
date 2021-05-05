// +build !windows

package probe_test

import (
	"context"
	"testing"
	"time"

	"github.com/macrat/ayd/store"
	"github.com/macrat/ayd/testutil"
)

func TestExecuteProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"exec:./testdata/test.sh?message=hello&code=0", store.STATUS_HEALTHY, "hello", ""},
		{"exec:./testdata/test.sh?message=world&code=1", store.STATUS_FAILURE, "world", ""},
		{"exec:echo#%0Ahello%0Aworld%0A%0A", store.STATUS_HEALTHY, "hello\nworld", ""},
		{"exec:./testdata/no-such-script", store.STATUS_UNKNOWN, ``, `exec: "./testdata/no-such-script": stat ./testdata/no-such-script: no such file or directory`},
		{"exec:./testdata/no-permission.sh", store.STATUS_UNKNOWN, ``, `exec: "./testdata/no-permission.sh": permission denied`},
		{"exec:no-such-command", store.STATUS_UNKNOWN, ``, `exec: "no-such-command": executable file not found in \$PATH`},
		{"exec:sleep#10", store.STATUS_UNKNOWN, `probe timed out`, ""},
		{"exec:echo#::status::unknown", store.STATUS_UNKNOWN, ``, ""},
		{"exec:echo#::status::failure", store.STATUS_FAILURE, ``, ""},
	})

	AssertTimeout(t, "exec:echo")
}

func BenchmarkExecuteProbe(b *testing.B) {
	p := testutil.NewProbe(b, "exec:echo#hello-world")

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Check(ctx, r)
	}
}
