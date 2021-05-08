package ayd

import (
	"errors"
	"io"
	"net/url"
	"os"
	"time"
)

var (
	// ErrEmptyTarget is the error means target URL of Record was empty
	ErrEmptyTarget = errors.New("the target URL is required")
)

// Logger is the logger for Ayd plugin
type Logger struct {
	writer   io.Writer
	target   *url.URL
	stime    time.Time
	latency  time.Duration
	useTimer bool
}

// NewLoggerWithWriter makes new Logger with a io.Writer
func NewLoggerWithWriter(w io.Writer, target *url.URL) Logger {
	return Logger{
		writer: w,
		target: target,
	}
}

// NewLogger makes new Logger
//
// This is the shorthand to `ayd.NewLoggerWithWriter(os.Stdout, target)`.
func NewLogger(target *url.URL) Logger {
	return NewLoggerWithWriter(os.Stdout, target)
}

// Print prints a Record
func (l Logger) Print(r Record) error {
	if r.Target == nil {
		if l.target == nil {
			return ErrEmptyTarget
		}
		r.Target = l.target
	}

	if l.useTimer {
		r.CheckedAt = l.stime
		r.Latency = time.Now().Sub(l.stime)
	} else {
		if r.CheckedAt.IsZero() {
			if l.stime.IsZero() {
				r.CheckedAt = time.Now()
			} else {
				r.CheckedAt = l.stime
			}
		}

		if r.Latency == 0 {
			r.Latency = l.latency
		}
	}

	_, err := l.writer.Write([]byte(r.String() + "\n"))
	return err
}

// Unknown prints Unknown status record
//
// Seealso StatusUnknown.
func (l Logger) Unknown(message string) error {
	return l.Print(Record{
		Status:  StatusUnknown,
		Message: message,
	})
}

// Healthy prints Healthy status record
//
// Seealso StatusHealthy.
func (l Logger) Healthy(message string) error {
	return l.Print(Record{
		Status:  StatusHealthy,
		Message: message,
	})
}

// Failure prints Failure status record
//
// Seealso StatusFailure.
func (l Logger) Failure(message string) error {
	return l.Print(Record{
		Status:  StatusFailure,
		Message: message,
	})
}

// Aborted prints Aborted status record
//
// Seealso StatusAborted.
func (l Logger) Aborted(message string) error {
	return l.Print(Record{
		Status:  StatusAborted,
		Message: message,
	})
}

// WithTarget makes new Logger with new target URL
func (l Logger) WithTarget(target *url.URL) Logger {
	return Logger{
		writer:   l.writer,
		target:   target,
		stime:    l.stime,
		latency:  l.latency,
		useTimer: l.useTimer,
	}
}

// WithTime makes new Logger with start time and latency value
func (l Logger) WithTime(startTime time.Time, latency time.Duration) Logger {
	return Logger{
		writer:  l.writer,
		target:  l.target,
		stime:   startTime,
		latency: latency,
	}
}

// StartTimer makes new Logger that set start time as current time, and start timer for latency from now.
//
// You can stop the timer with StopTimer method, or just call print method like Healthy or Failure.
func (l Logger) StartTimer() Logger {
	return Logger{
		writer:   l.writer,
		target:   l.target,
		stime:    time.Now(),
		useTimer: true,
	}
}

// StopTimer stops latency timer that started by StartTimer method, and makes new Logger with measured latency.
func (l Logger) StopTimer() Logger {
	return Logger{
		writer:  l.writer,
		target:  l.target,
		stime:   l.stime,
		latency: time.Now().Sub(l.stime),
	}
}
