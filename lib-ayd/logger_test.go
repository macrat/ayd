package ayd_test

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/macrat/ayd/lib-ayd"
)

func ExampleLogger() {
	target, _ := url.Parse("foobar:your-plugin-url")

	logger := ayd.NewLogger(target)

	logger.Healthy("hello world")
}

func ExampleLogger_setExtraValues() {
	logger := ayd.NewLogger(nil)

	// change target URL
	target, _ := url.Parse("foobar:your-plugin-url")
	logger = logger.WithTarget(target)

	// set check time and latency of the target
	startTime, _ := time.Parse(time.RFC3339, "2001-02-03T16:05:06+09:00")
	latency := 123 * time.Millisecond
	logger = logger.WithTime(startTime, latency)

	// report target status with a message
	logger.Healthy("target is healthy")
	logger.Failure("target seems down")
	logger.Unknown("failed to check, so target status is unknown")
	logger.Aborted("the check was aborted by user or something")

	// Output:
	// 2001-02-03T16:05:06+09:00	HEALTHY	123.000	foobar:your-plugin-url	target is healthy
	// 2001-02-03T16:05:06+09:00	FAILURE	123.000	foobar:your-plugin-url	target seems down
	// 2001-02-03T16:05:06+09:00	UNKNOWN	123.000	foobar:your-plugin-url	failed to check, so target status is unknown
	// 2001-02-03T16:05:06+09:00	ABORTED	123.000	foobar:your-plugin-url	the check was aborted by user or something
}

func ExampleLogger_Print() {
	logger := ayd.NewLogger(nil)

	logger.Print(ayd.Record{
		Target:    &url.URL{Scheme: "foo", Host: "bar"},
		Status:    ayd.StatusHealthy,
		CheckedAt: time.Date(2001, 2, 3, 16, 5, 6, 7, time.UTC),
		Message:   "hello world",
	})

	logger.Print(ayd.Record{
		Target:    &url.URL{Scheme: "foo", Host: "bar"},
		CheckedAt: time.Date(2001, 2, 3, 16, 5, 7, 0, time.UTC),
		Message:   "without status",
	})

	err := logger.Print(ayd.Record{
		CheckedAt: time.Date(2001, 2, 3, 16, 5, 8, 0, time.UTC),
		Message:   "without target",
	})
	fmt.Println("error:", err)

	// Output:
	// 2001-02-03T16:05:06Z	HEALTHY	0.000	foo://bar	hello world
	// 2001-02-03T16:05:07Z	UNKNOWN	0.000	foo://bar	without status
	// error: invalid record: the target URL is required
}

func ExampleLogger_WithTarget() {
	logger := ayd.NewLogger(nil)

	target, _ := url.Parse("foobar:your-plugin-url")

	logger.WithTarget(target).Healthy("hello world")
}

func ExampleLogger_WithTime() {
	target, _ := url.Parse("foobar:your-plugin-url")

	logger := ayd.NewLogger(target)

	startTime, _ := time.Parse(time.RFC3339, "2001-02-03T16:05:06+09:00")
	latency := 123 * time.Millisecond

	logger.WithTime(startTime, latency).Healthy("hello world")

	// Output:
	// 2001-02-03T16:05:06+09:00	HEALTHY	123.000	foobar:your-plugin-url	hello world
}

func ExampleLogger_StartTimer() {
	target, _ := url.Parse("foobar:your-plugin-url")

	logger := ayd.NewLogger(target)

	l := logger.StartTimer()
	// check your target status
	l.Healthy("hello world")
}

func ExampleLogger_StopTimer() {
	target, _ := url.Parse("foobar:your-plugin-url")

	logger := ayd.NewLogger(target)

	l := logger.StartTimer()
	// check your target status
	l = l.StopTimer()

	// do something, for example calculate result of the check

	l.Healthy("hello world")
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

	target, _ := url.Parse("dummy:")
	l := ayd.NewLoggerWithWriter(buf, target)

	l.Healthy("hello")
	assert("^[-+:0-9TZ]+\tHEALTHY\t0\\.000\tdummy:\thello\n$")

	l.Failure("world")
	assert("^[-+:0-9TZ]+\tFAILURE\t0\\.000\tdummy:\tworld\n$")

	l.StartTimer().Healthy("no-delay")
	assert("^[-+:0-9TZ]+\tHEALTHY\t0\\.[0-9]{3}\tdummy:\tno-delay\n$")

	l.StartTimer().StopTimer().Healthy("no-delay-stop")
	assert("^[-+:0-9TZ]+\tHEALTHY\t0\\.[0-9]{3}\tdummy:\tno-delay-stop\n$")

	l2 := l.StartTimer()
	time.Sleep(100 * time.Millisecond)
	l2.Healthy("no-delay")
	assert("^[-+:0-9TZ]+\tHEALTHY\t[0-9]{3}\\.[0-9]{3}\tdummy:\tno-delay\n$")
}
