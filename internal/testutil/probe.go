package testutil

import (
	"testing"

	"github.com/macrat/ayd/internal/scheme"
)

func NewProber(t testing.TB, u string) scheme.Prober {
	t.Helper()

	p, err := scheme.NewProber(u)
	if err != nil {
		t.Fatalf("failed to create probe: %s", err)
	}

	return p
}
