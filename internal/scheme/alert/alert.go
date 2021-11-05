package alert

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	api "github.com/macrat/ayd/lib-ayd"
)

// Reporter is a shorthand to ayd/internal/scheme.Reporter.
type Reporter = scheme.Reporter

type Alert interface {
	Trigger(context.Context, api.Record, Reporter)
}

func New(target string) (Alert, error) {
	p, err := scheme.WithoutPluginProbe(scheme.NewProbe(target))
	if err == scheme.ErrUnsupportedScheme {
		return NewPluginAlert(target)
	} else if err != nil {
		return nil, err
	}

	return ProbeAlert{p.Target()}, nil
}

type ReplaceReporter struct {
	Target   *url.URL
	Upstream Reporter
}

func (r ReplaceReporter) Report(_ *url.URL, rec api.Record) {
	scheme, _, _ := scheme.SplitScheme(rec.Target.Scheme)

	if scheme != "alert" && scheme != "ayd" {
		rec.Target = r.Target
	}
	r.Upstream.Report(r.Target, rec)
}

func (r ReplaceReporter) DeactivateTarget(source *url.URL, targets ...*url.URL) {
	r.Upstream.DeactivateTarget(source, targets...)
}

type ProbeAlert struct {
	target *url.URL
}

func (a ProbeAlert) Trigger(ctx context.Context, lastRecord api.Record, r Reporter) {
	qs := a.target.Query()
	qs.Set("ayd_checked_at", lastRecord.CheckedAt.Format(time.RFC3339))
	qs.Set("ayd_status", lastRecord.Status.String())
	qs.Set("ayd_latency", strconv.FormatFloat(float64(lastRecord.Latency.Microseconds())/1000.0, 'f', -1, 64))
	qs.Set("ayd_target", lastRecord.Target.String())
	qs.Set("ayd_message", lastRecord.Message)

	u := *a.target
	u.RawQuery = qs.Encode()

	reporter := ReplaceReporter{
		&url.URL{Scheme: "alert", Opaque: a.target.String()},
		r,
	}

	p, err := scheme.WithoutPluginProbe(scheme.NewProbeFromURL(&u))
	if err != nil {
		reporter.Report(reporter.Target, api.Record{
			CheckedAt: time.Now(),
			Status:    api.StatusUnknown,
			Message:   err.Error(),
		})
		return
	}

	p.Check(ctx, reporter)
}

type AlertReporter struct {
	Source   *url.URL
	Upstream scheme.Reporter
}

func (r AlertReporter) Report(_ *url.URL, rec api.Record) {
	scheme, _, _ := scheme.SplitScheme(rec.Target.Scheme)

	if scheme != "alert" && scheme != "ayd" {
		rec.Target = &url.URL{
			Scheme: "alert",
			Opaque: rec.Target.String(),
		}
	}
	r.Upstream.Report(r.Source, rec)
}

func (r AlertReporter) DeactivateTarget(source *url.URL, targets ...*url.URL) {
	r.Upstream.DeactivateTarget(source, targets...)
}

type PluginAlert struct {
	target *url.URL
}

func NewPluginAlert(target string) (PluginAlert, error) {
	u, err := url.Parse(target)
	if err != nil {
		return PluginAlert{}, err
	}

	if s, _, _ := scheme.SplitScheme(u.Scheme); s == "ayd" || s == "alert" {
		return PluginAlert{}, scheme.ErrUnsupportedScheme
	}

	p := PluginAlert{
		target: u,
	}

	if _, err := scheme.FindPlugin(u.Scheme, "alert"); err != nil {
		return PluginAlert{}, err
	}

	return p, nil
}

func (a PluginAlert) Trigger(ctx context.Context, lastRecord api.Record, r Reporter) {
	scheme.ExecutePlugin(
		ctx,
		AlertReporter{&url.URL{Scheme: "alert", Opaque: a.target.String()}, r},
		"alert",
		a.target,
		[]string{
			a.target.String(),
			lastRecord.CheckedAt.Format(time.RFC3339),
			lastRecord.Status.String(),
			strconv.FormatFloat(float64(lastRecord.Latency.Microseconds())/1000.0, 'f', -1, 64),
			lastRecord.Target.String(),
			lastRecord.Message,
		},
		os.Environ(),
	)
}
