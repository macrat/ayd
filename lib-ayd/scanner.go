package ayd

import (
	"io"
	"bufio"
	"time"
)

// LogScanner is the interface to read ayd's log format.
type LogScanner interface {
	// Close closes the scanner.
	Close() error

	// Scan scans the next line of log. If there is no more log, it returns false.
	Scan() bool

	// Record returns current record.
	Record() Record
}

type fileScanner struct {
	file    io.ReadCloser
	scanner *bufio.Scanner
	since   time.Time
	until   time.Time
	rec     Record
}

// NewLogScanner creates a new LogScanner from io.ReadCloser.
func NewLogScanner(f io.ReadCloser) *fileScanner {
	return NewLogScannerWithPeriod(f, time.Time{}, time.Unix(2<<61, 0))
}

// NewLogScannerWithPeriod creates a new LogScanner from io.ReadCloser, with period specification.
func NewLogScannerWithPeriod(f io.ReadCloser, since, until time.Time) *fileScanner {
	return &fileScanner{
		file:    f,
		scanner: bufio.NewScanner(f),
		since:   since,
		until:   until,
	}
}

func (r *fileScanner) Close() error {
	return r.file.Close()
}

func (r *fileScanner) Scan() bool {
	for r.scanner.Scan() {
		rec, err := ParseRecord(r.scanner.Text())
		if err != nil || rec.CheckedAt.Before(r.since) {
			continue
		}
		if !r.until.After(rec.CheckedAt) {
			return false
		}
		r.rec = rec
		return true
	}
	return false
}

func (r *fileScanner) Record() Record {
	return r.rec
}
