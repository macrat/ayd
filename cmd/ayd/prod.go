//go:build !debug
// +build !debug

package main

import (
	"github.com/macrat/ayd/internal/store"
)

// startDebugLogger starts debug logger and pprof server.
// But this function nothing to do in production mode.
// Please see also debug.go
func startDebugLogger(s *store.Store) {
	// nothing to do.
}
