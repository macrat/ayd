package scheme

import (
	"context"
	"errors"
	"net/url"
	"sync"

	"github.com/macrat/ayd/internal/ayderr"
	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrInvalidAlertURL = errors.New("invalid alert URL")
)

type Alerter interface {
	Target() *url.URL
	Alert(context.Context, Reporter, api.Record)
}

func NewAlerterFromURL(u *url.URL) (Alerter, error) {
	scheme, _, _ := SplitScheme(u.Scheme)

	switch scheme {
	case "http", "https":
		return NewHTTPScheme(u)
	case "ping", "ping4", "ping6":
		return NewPingScheme(u)
	case "tcp", "tcp4", "tcp6":
		return NewTCPScheme(u)
	case "dns", "dns4", "dns6":
		return NewDNSScheme(u)
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
	u, err := url.Parse(target)
	if err != nil {
		return nil, ErrInvalidURL
	}

	return NewAlerterFromURL(u)
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

// AlerterSet is a set of alerts.
// It also implements Alerter alertinterface.
type AlerterSet []Alerter

func NewAlerterSet(targets []string) (AlerterSet, error) {
	alerts := make(AlerterSet, len(targets))
	errs := &ayderr.ListBuilder{What: ErrInvalidAlertURL}

	for i, t := range targets {
		var err error
		alerts[i], err = NewAlerter(t)
		if err != nil {
			errs.Pushf("%s: %w", t, err)
		}
	}

	return alerts, errs.Build()
}

// Target implements Alert interface.
// This method always returns alert-set: URL.
func (as AlerterSet) Target() *url.URL {
	return &url.URL{Scheme: "alert-set"}
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
