package scheme

import (
	"context"
	"errors"
	"net/url"

	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrInvalidURL        = errors.New("invalid URL")
	ErrMissingScheme     = errors.New("missing scheme in URL")
	ErrUnsupportedScheme = errors.New("unsupported scheme")
	ErrMissingHost       = errors.New("missing target host")
)

type Prober interface {
	Target() *url.URL
	Probe(context.Context, Reporter)
}

func NewProberFromURL(u *url.URL) (Prober, error) {
	scheme, _, _ := SplitScheme(u.Scheme)

	switch scheme {
	case "http", "https":
		return NewHTTPProbe(u)
	case "ping", "ping4", "ping6":
		return NewPingScheme(u)
	case "tcp", "tcp4", "tcp6":
		return NewTCPScheme(u)
	case "dns", "dns4", "dns6":
		return NewDNSProbe(u)
	case "exec":
		return NewExecuteProbe(u)
	case "source":
		return NewSourceProbe(u)
	case "dummy":
		return NewDummyScheme(u)
	default:
		return NewPluginProbe(u)
	}
}

func NewProber(rawURL string) (Prober, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, ErrInvalidURL
	}

	if u.Scheme == "" {
		return nil, ErrMissingScheme
	}

	return NewProberFromURL(u)
}

func WithoutPluginProbe(p Prober, err error) (Prober, error) {
	if err != nil {
		return nil, err
	}

	if _, ok := p.(PluginScheme); ok {
		return nil, ErrUnsupportedScheme
	}

	return p, nil
}

func timeoutOr(ctx context.Context, r api.Record) api.Record {
	switch ctx.Err() {
	case context.Canceled:
		r.Status = api.StatusAborted
		r.Message = "probe aborted"
	case context.DeadlineExceeded:
		r.Status = api.StatusFailure
		r.Message = "probe timed out"
	default:
	}
	return r
}
