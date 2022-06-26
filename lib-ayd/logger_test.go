package ayd_test

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/macrat/ayd/lib-ayd"
)

func ExampleLogger() {
	target, _ := ayd.ParseURL("foobar:your-plugin-url")

	logger := ayd.NewLogger(target)

	logger.Healthy("hello world", nil)
	logger.Failure("hello world", map[string]interface{}{
		"extra": "information",
	})
}

func ExampleLogger_setExtraValues() {
	logger := ayd.NewLogger(nil)

	// change target URL
	target, _ := ayd.ParseURL("foobar:your-plugin-url")
	logger = logger.WithTarget(target)

	// set check time and latency of the target
	startTime, _ := time.Parse(time.RFC3339, "2001-02-03T16:05:06+09:00")
	latency := 123 * time.Millisecond
	logger = logger.WithTime(startTime, latency)

	// report target status with a message
	logger.Healthy("target is healthy", map[string]interface{}{
		"hello": "world",
	})
	logger.Degrade("target is partialy working", map[string]interface{}{
		"number": 42,
	})
	logger.Failure("target seems down", map[string]interface{}{
		"list": []string{"hello", "world"},
	})
	logger.Unknown("failed to check, so target status is unknown", map[string]interface{}{
		"obj": map[string]string{"hello": "world"},
	})
	logger.Aborted("the check was aborted by user or something", nil)

	// Output:
	// {"time":"2001-02-03T16:05:06+09:00", "status":"HEALTHY", "latency":123.000, "target":"foobar:your-plugin-url", "message":"target is healthy", "hello":"world"}
	// {"time":"2001-02-03T16:05:06+09:00", "status":"DEGRADE", "latency":123.000, "target":"foobar:your-plugin-url", "message":"target is partialy working", "number":42}
	// {"time":"2001-02-03T16:05:06+09:00", "status":"FAILURE", "latency":123.000, "target":"foobar:your-plugin-url", "message":"target seems down", "list":["hello","world"]}
	// {"time":"2001-02-03T16:05:06+09:00", "status":"UNKNOWN", "latency":123.000, "target":"foobar:your-plugin-url", "message":"failed to check, so target status is unknown", "obj":{"hello":"world"}}
	// {"time":"2001-02-03T16:05:06+09:00", "status":"ABORTED", "latency":123.000, "target":"foobar:your-plugin-url", "message":"the check was aborted by user or something"}
}

func ExampleLogger_Print() {
	logger := ayd.NewLogger(nil)

	logger.Print(ayd.Record{
		Target:  &ayd.URL{Scheme: "foo", Host: "bar"},
		Status:  ayd.StatusHealthy,
		Time:    time.Date(2001, 2, 3, 16, 5, 6, 7, time.UTC),
		Message: "hello world",
	})

	logger.Print(ayd.Record{
		Target:  &ayd.URL{Scheme: "foo", Host: "bar"},
		Time:    time.Date(2001, 2, 3, 16, 5, 7, 0, time.UTC),
		Message: "without status",
	})

	logger.Print(ayd.Record{
		Target:  &ayd.URL{Scheme: "foo", Host: "bar"},
		Status:  ayd.StatusHealthy,
		Time:    time.Date(2001, 2, 3, 16, 5, 6, 7, time.UTC),
		Message: "with extra",
		Extra: map[string]interface{}{
			"hello": "world",
		},
	})

	err := logger.Print(ayd.Record{
		Time:    time.Date(2001, 2, 3, 16, 5, 8, 0, time.UTC),
		Message: "without target",
	})
	fmt.Println("error:", err)

	// Output:
	// {"time":"2001-02-03T16:05:06Z", "status":"HEALTHY", "latency":0.000, "target":"foo://bar", "message":"hello world"}
	// {"time":"2001-02-03T16:05:07Z", "status":"UNKNOWN", "latency":0.000, "target":"foo://bar", "message":"without status"}
	// {"time":"2001-02-03T16:05:06Z", "status":"HEALTHY", "latency":0.000, "target":"foo://bar", "message":"with extra", "hello":"world"}
	// error: invalid record: the target URL is required
}

func ExampleLogger_WithTarget() {
	logger := ayd.NewLogger(nil)

	target, _ := ayd.ParseURL("foobar:your-plugin-url")

	logger.WithTarget(target).Healthy("hello world", nil)
}

func ExampleLogger_WithTime() {
	target, _ := ayd.ParseURL("foobar:your-plugin-url")

	logger := ayd.NewLogger(target)

	startTime, _ := time.Parse(time.RFC3339, "2001-02-03T16:05:06+09:00")
	latency := 123 * time.Millisecond

	logger.WithTime(startTime, latency).Healthy("hello world", nil)

	// Output:
	// {"time":"2001-02-03T16:05:06+09:00", "status":"HEALTHY", "latency":123.000, "target":"foobar:your-plugin-url", "message":"hello world"}
}

func ExampleLogger_StartTimer() {
	target, _ := ayd.ParseURL("foobar:your-plugin-url")

	logger := ayd.NewLogger(target)

	l := logger.StartTimer()
	// check your target status
	l.Healthy("hello world", nil)
}

func ExampleLogger_StopTimer() {
	target, _ := ayd.ParseURL("foobar:your-plugin-url")

	logger := ayd.NewLogger(target)

	l := logger.StartTimer()
	// check your target status
	l = l.StopTimer()

	// do something, for example calculate result of the check

	l.Healthy("hello world", nil)
}

func TestLogger_Print(t *testing.T) {
	buf := &bytes.Buffer{}

	assert := func(pattern string) {
		t.Helper()

		if ok, _ := regexp.Match(pattern, buf.Bytes()); !ok {
			t.Errorf("expected log matches with %q but got:\n%s", pattern, buf)
		}

		buf.Reset()
	}

	target, _ := ayd.ParseURL("dummy:")
	l := ayd.NewLoggerWithWriter(buf, target)

	l.Healthy("hello", nil)
	assert(`^{"time":"[-+:0-9TZ]+", "status":"HEALTHY", "latency":0\.000, "target":"dummy:", "message":"hello"}` + "\n$")

	l.Failure("world", nil)
	assert(`^{"time":"[-+:0-9TZ]+", "status":"FAILURE", "latency":0\.000, "target":"dummy:", "message":"world"}` + "\n$")

	l.StartTimer().Healthy("no-delay", nil)
	assert(`^{"time":"[-+:0-9TZ]+", "status":"HEALTHY", "latency":0\.[0-9]{3}, "target":"dummy:", "message":"no-delay"}` + "\n$")

	l.StartTimer().StopTimer().Healthy("no-delay-stop", nil)
	assert(`^{"time":"[-+:0-9TZ]+", "status":"HEALTHY", "latency":0\.[0-9]{3}, "target":"dummy:", "message":"no-delay-stop"}` + "\n$")

	l2 := l.StartTimer()
	time.Sleep(100 * time.Millisecond)
	l2.Healthy("with-delay", nil)
	assert(`^{"time":"[-+:0-9TZ]+", "status":"HEALTHY", "latency":[0-9]{3}\.[0-9]{3}, "target":"dummy:", "message":"with-delay"}` + "\n$")
}
