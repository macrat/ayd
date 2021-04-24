package probe_test

import (
	"testing"

	"github.com/macrat/ayd/store"
)

func TestPingProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"ping:localhost", store.STATUS_HEALTHY, `rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/rcv=4/4`},
	})

	AssertTimeout(t, "ping:localhost")
}
