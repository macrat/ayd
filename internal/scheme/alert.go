package scheme

import (
	"context"
	"net/url"
	"strconv"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type Alerter interface {
	Alert(context.Context, Reporter, api.Record)
}

func NewAlerter(target string) (Alerter, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, ErrInvalidURL
	}

	p, err := WithoutPluginProbe(NewProberFromURL(u))
	if err == ErrUnsupportedScheme {
		return NewPluginAlert(u)
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
	if s, _, _ := SplitScheme(rec.Target.Scheme); s != "alert" && s != "ayd" {
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

func (a ProbeAlert) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
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

	p, err := WithoutPluginProbe(NewProberFromURL(&u))
	if err != nil {
		reporter.Report(reporter.Target, api.Record{
			CheckedAt: time.Now(),
			Status:    api.StatusUnknown,
			Message:   err.Error(),
		})
		return
	}

	p.Probe(ctx, reporter)
}

type AlertReporter struct {
	Source   *url.URL
	Upstream Reporter
}

func (r AlertReporter) Report(_ *url.URL, rec api.Record) {
	if s, _, _ := SplitScheme(rec.Target.Scheme); s != "alert" && s != "ayd" {
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
