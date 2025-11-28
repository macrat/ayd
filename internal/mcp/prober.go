package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	api "github.com/macrat/ayd/lib-ayd"
)

// DefaultProber implements Prober interface using scheme.NewProber.
type DefaultProber struct{}

// NewDefaultProber creates a new DefaultProber.
func NewDefaultProber() *DefaultProber {
	return &DefaultProber{}
}

// Probe executes a probe on the given target URL and returns the result.
func (p *DefaultProber) Probe(ctx context.Context, targetURL string) api.Record {
	prober, err := scheme.NewProber(targetURL)
	if err != nil {
		target, _ := api.ParseURL(targetURL)
		return api.Record{
			Time:    time.Now(),
			Status:  api.StatusUnknown,
			Target:  target,
			Message: fmt.Sprintf("failed to create prober: %s", err),
		}
	}

	reporter := &singleRecordReporter{}
	prober.Probe(ctx, reporter)

	if reporter.record != nil {
		return *reporter.record
	}

	return api.Record{
		Time:    time.Now(),
		Status:  api.StatusUnknown,
		Target:  prober.Target(),
		Message: "no result",
	}
}

// singleRecordReporter captures a single Record from a probe.
type singleRecordReporter struct {
	record *api.Record
}

func (r *singleRecordReporter) Report(source *api.URL, rec api.Record) {
	if r.record == nil {
		r.record = &rec
	}
}

func (r *singleRecordReporter) DeactivateTarget(source *api.URL, targets ...*api.URL) {
	// No-op for single record reporter
}
