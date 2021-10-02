package testutil

import (
	"testing"

	"github.com/macrat/ayd/internal/probe"
)

func NewProbe(t testing.TB, u string) probe.Probe {
	t.Helper()

	p, err := probe.New(u)
	if err != nil {
		t.Fatalf("failed to create probe: %s", err)
	}

	return p
}
