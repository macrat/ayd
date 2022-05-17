package scheme

import (
	"context"
	"errors"
	"sync"

	"github.com/macrat/ayd/internal/ayderr"
	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrInvalidAlertURL        = errors.New("invalid alert URL")
	ErrUnsupportedAlertScheme = errors.New("unsupported scheme for alert")
)

// Alerter is the interface to send alerts to somewhere.
type Alerter interface {
	// Target returns the alert target URL.
	// This URL should not change during lifetime of the instance.
	Target() *api.URL

	// Alert sends an alert to the target, and report result(s) to the Reporter.
	Alert(context.Context, Reporter, api.Record)
}

func NewAlerterFromURL(u *api.URL) (Alerter, error) {
	scheme, _, _ := SplitScheme(u.Scheme)

	switch scheme {
	case "http", "https":
		return NewHTTPScheme(u)
	case "ftp", "ftps":
		return nil, ErrUnsupportedAlertScheme
	case "ping", "ping4", "ping6":
		return nil, ErrUnsupportedAlertScheme
	case "tcp", "tcp4", "tcp6":
		return nil, ErrUnsupportedAlertScheme
	case "dns", "dns4", "dns6":
		return nil, ErrUnsupportedAlertScheme
	case "exec":
		return NewExecScheme(u)
	case "source":
		return NewSourceAlert(u)
	case "dummy":
		return NewDummyScheme(u)
	default:
		return NewPluginAlert(u)
	}
}

func NewAlerter(target string) (Alerter, error) {
	u, err := api.ParseURL(target)
	if err != nil {
		return nil, ErrInvalidURL
	}

	return NewAlerterFromURL(u)
}

// AlertReporter is a wrapper of Reporter interface for alert schemes.
// It replaces source URL, and puts "alert:" prefix to the target URL.
type AlertReporter struct {
	Source   *api.URL
	Upstream Reporter
}

func (r AlertReporter) Report(_ *api.URL, rec api.Record) {
	if s, _, _ := SplitScheme(rec.Target.Scheme); s != "alert" && s != "ayd" {
		rec.Target = &api.URL{
			Scheme: "alert",
			Opaque: rec.Target.String(),
		}
	}
	r.Upstream.Report(r.Source, rec)
}

func (r AlertReporter) DeactivateTarget(source *api.URL, targets ...*api.URL) {
	r.Upstream.DeactivateTarget(source, targets...)
}

// AlerterSet is a set of alerts.
// It also implements Alerter alertinterface.
type AlerterSet []Alerter

func NewAlerterSet(targets []string) (AlerterSet, error) {
	urls := &urlSet{}
	alerts := make(AlerterSet, 0, len(targets))
	errs := &ayderr.ListBuilder{What: ErrInvalidAlertURL}

	for _, t := range targets {
		a, err := NewAlerter(t)
		if err != nil {
			errs.Pushf("%s: %w", t, err)
		} else if !urls.Has(a.Target()) {
			urls.Add(a.Target())
			alerts = append(alerts, a)
		}
	}

	return alerts, errs.Build()
}

// Target implements Alert interface.
// This method always returns alert-set: URL.
func (as AlerterSet) Target() *api.URL {
	return &api.URL{Scheme: "alert-set"}
}

// Alert of AlerterSet calls all Alert methods of children parallelly.
// This method blocks until all alerts done.
func (as AlerterSet) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	wg := &sync.WaitGroup{}

	for _, a := range as {
		wg.Add(1)
		go func(a Alerter) {
			a.Alert(ctx, r, lastRecord)
			wg.Done()
		}(a)
	}

	wg.Wait()
}
