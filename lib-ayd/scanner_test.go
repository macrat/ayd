package ayd_test

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/macrat/ayd/lib-ayd"
)

func ExampleNewLogScanner() {
	f := io.NopCloser(strings.NewReader(`
{"time":"2001-02-03T16:05:06Z", "status":"HEALTHY", "latency":1.234, "target":"https://example.com", "message":"200 OK"}
{"time":"2001-02-03T16:15:06Z", "status":"HEALTHY", "latency":2.345, "target":"https://example.com", "message":"200 OK"}
{"time":"2001-02-03T16:25:06Z", "status":"HEALTHY", "latency":3.456, "target":"https://example.com", "message":"200 OK"}
`))

	s := ayd.NewLogScanner(f)
	defer s.Close()

	for s.Scan() {
		fmt.Println(s.Record().Time)
	}

	// OUTPUT:
	// 2001-02-03 16:05:06 +0000 UTC
	// 2001-02-03 16:15:06 +0000 UTC
	// 2001-02-03 16:25:06 +0000 UTC
}

func ExampleNewLogScannerWithPeriod() {
	f := io.NopCloser(strings.NewReader(`
{"time":"2001-02-03T16:05:06Z", "status":"HEALTHY", "latency":1.234, "target":"https://example.com", "message":"200 OK"}
{"time":"2001-02-03T16:15:06Z", "status":"HEALTHY", "latency":2.345, "target":"https://example.com", "message":"200 OK"}
{"time":"2001-02-03T16:25:06Z", "status":"HEALTHY", "latency":3.456, "target":"https://example.com", "message":"200 OK"}
`))

	s := ayd.NewLogScannerWithPeriod(
		f,
		time.Date(2001, 2, 3, 16, 7, 0, 0, time.UTC),
		time.Date(2001, 2, 3, 16, 25, 0, 0, time.UTC),
	)
	defer s.Close()

	for s.Scan() {
		fmt.Println(s.Record().Time)
	}

	// OUTPUT:
	// 2001-02-03 16:15:06 +0000 UTC
}
