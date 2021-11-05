package testutil

import (
	"testing"

	"github.com/macrat/ayd/internal/scheme"
)

func NewProbe(t testing.TB, u string) scheme.Probe {
	t.Helper()

	p, err := scheme.NewProbe(u)
	if err != nil {
		t.Fatalf("failed to create probe: %s", err)
	}

	return p
}
