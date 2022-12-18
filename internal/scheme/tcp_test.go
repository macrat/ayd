package scheme_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestTCPScheme_Probe(t *testing.T) {
	t.Parallel()

	server := RunDummyHTTPServer()
	defer server.Close()

	AssertProbe(t, []ProbeTest{
		{strings.Replace(server.URL, "http://", "tcp://", 1), api.StatusHealthy, "succeed to connect\n---\nsource_addr: ([0-9.]+|\\[[0-9a-fA-F:]+\\]):[0-9]+\ntarget_addr: (127.0.0.1|\\[::\\]):[0-9]+", ""},

		{"tcp://localhost", api.StatusUnknown, ``, "TCP target's port number is required"},
	}, 10)
}

func BenchmarkTCPScheme(b *testing.B) {
	server := RunDummyHTTPServer()
	defer server.Close()

	p := testutil.NewProber(b, strings.Replace(server.URL, "http:", "tcp:", 1))

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Probe(ctx, r)
	}
}
