//go:build debug
// +build debug

package main

import (
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
)

// startDebugLogger starts debug logger and pprof server.
func startDebugLogger(s *store.Store) {
	uptime := time.Now()

	scope := &api.URL{Scheme: "ayd", Opaque: "debug"}
	report := func(latency time.Duration, message string, extra map[string]interface{}) {
		s.Report(scope, api.Record{
			Time:    time.Now(),
			Status:  api.StatusHealthy,
			Latency: latency,
			Target:  scope,
			Message: message,
			Extra:   extra,
		})
	}

	go func() {
		report(time.Since(uptime), "start in debug mode", map[string]interface{}{
			"arch":      runtime.GOARCH,
			"compiler":  runtime.Compiler,
			"os":        runtime.GOOS,
			"goversion": runtime.Version(),
		})

		processStatus := func() {
			var mem runtime.MemStats

			stime := time.Now()
			runtime.ReadMemStats(&mem)
			report(time.Since(stime), "process status", map[string]interface{}{
				"num_goroutine":  runtime.NumGoroutine(),
				"heap_alloc":     mem.HeapAlloc,
				"mallocs":        mem.Mallocs,
				"mem_frees":      mem.Frees,
				"num_gc":         mem.NumGC,
				"uptime_seconds": time.Since(uptime).Seconds(),
			})
		}

		processStatus()

		t := time.Tick(5 * time.Second)
		for range t {
			processStatus()
		}
	}()

	go func() {
		report(0, "start pprof server", map[string]interface{}{
			"url": "http://localhost:6060",
		})
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			report(0, "pprof server has stopped", map[string]interface{}{
				"reason": err.Error(),
			})
		}
	}()
}
