// +build debug

package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

func debugStatusReporter() {
	f, err := os.OpenFile("ayd_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG: failed to open debug log file: %s\n", err)
		return
	}
	fmt.Fprintf(f, "timestamp\tcurrent_goroutines\tcurrent_heap_bytes\tnum_mallocs\tnum_frees\tnum_gc\n")

	var mem runtime.MemStats

	t := time.Tick(5 * time.Second)
	for range t {
		runtime.ReadMemStats(&mem)
		fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\n", time.Now().Format(time.RFC3339), runtime.NumGoroutine(), mem.HeapAlloc, mem.Mallocs, mem.Frees, mem.NumGC)
	}
}

func init() {
	go debugStatusReporter()
}
