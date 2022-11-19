package ayd

import (
	"io"
	"os"
	"time"

	"github.com/macrat/ayd/internal/ayderr"
)

// Logger is the logger for Ayd plugin
type Logger struct {
	writer   io.Writer
	target   *URL
	stime    time.Time
	latency  time.Duration
	useTimer bool
}

// NewLoggerWithWriter makes new Logger with a io.Writer
func NewLoggerWithWriter(w io.Writer, target *URL) Logger {
	return Logger{
		writer: w,
		target: target,
	}
}

// NewLogger makes new Logger
//
// This is the shorthand to `ayd.NewLoggerWithWriter(os.Stdout, target)`.
func NewLogger(target *URL) Logger {
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
		r.Time = l.stime
		r.Latency = time.Since(l.stime)
	} else {
		if r.Time.IsZero() {
			if l.stime.IsZero() {
				r.Time = time.Now()
			} else {
				r.Time = l.stime
			}
		}

		if r.Latency == 0 {
			r.Latency = l.latency
		}
	}

	if r.Latency < 0 {
		r.Latency = 0
	}

	_, err := l.writer.Write([]byte(r.String() + "\n"))
	if err != nil {
		return ayderr.New(ErrIO, err, "failed to write log")
	}
	return nil
}

// Healthy prints Healthy status record.
//
// Seealso StatusHealthy.
func (l Logger) Healthy(message string, extra map[string]interface{}) error {
	return l.Print(Record{
		Status:  StatusHealthy,
		Message: message,
		Extra:   extra,
	})
}

// Aborted prints Aborted status record.
//
// Seealso StatusAborted.
func (l Logger) Aborted(message string, extra map[string]interface{}) error {
	return l.Print(Record{
		Status:  StatusAborted,
		Message: message,
		Extra:   extra,
	})
}

// Unknown prints Unknown status record.
//
// Seealso StatusUnknown.
func (l Logger) Unknown(message string, extra map[string]interface{}) error {
	return l.Print(Record{
		Status:  StatusUnknown,
		Message: message,
		Extra:   extra,
	})
}

// Degrade prints Degrade status record.
//
// Seealso StatusDegrade.
func (l Logger) Degrade(message string, extra map[string]interface{}) error {
	return l.Print(Record{
		Status:  StatusDegrade,
		Message: message,
		Extra:   extra,
	})
}

// Failure prints Failure status record.
//
// Seealso StatusFailure.
func (l Logger) Failure(message string, extra map[string]interface{}) error {
	return l.Print(Record{
		Status:  StatusFailure,
		Message: message,
		Extra:   extra,
	})
}

// WithTarget makes new Logger with new target URL
func (l Logger) WithTarget(target *URL) Logger {
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
		latency: time.Since(l.stime),
	}
}
