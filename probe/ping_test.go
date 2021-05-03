package probe_test

import (
	"context"
	"testing"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

func TestPingProbe(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := probe.StartPinger(ctx); err != nil {
		t.Fatalf("failed to start pinger: %s", err)
	}

	AssertProbe(t, []ProbeTest{
		{"ping:localhost", store.STATUS_HEALTHY, `rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/rcv=4/4`, ""},
		{"ping:127.0.0.1", store.STATUS_HEALTHY, `rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/rcv=4/4`, ""},
		{"ping:::1", store.STATUS_HEALTHY, `rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/rcv=4/4`, ""},
		{"ping:of-course-definitely-no-such-host", store.STATUS_UNKNOWN, `.*`, ""},
	})

	AssertTimeout(t, "ping:localhost")
}
