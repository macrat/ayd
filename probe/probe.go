package probe

import (
	"context"
	"errors"
	"net/url"
	"strings"

	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrInvalidURL        = errors.New("invalid URL")
	ErrMissingScheme     = errors.New("missing scheme in URL")
	ErrUnsupportedScheme = errors.New("unsupported scheme")
)

type Reporter interface {
	Report(r api.Record)
}

type Probe interface {
	Target() *url.URL
	Check(context.Context, Reporter)
}

func NewFromURL(u *url.URL) (Probe, error) {
	if strings.HasPrefix(u.Scheme, "http-") || strings.HasPrefix(u.Scheme, "https-") {
		return NewHTTPProbe(u)
	}

	switch u.Scheme {
	case "http", "https":
		return NewHTTPProbe(u)
	case "ping":
		return NewPingProbe(u)
	case "tcp", "tcp4", "tcp6":
		return NewTCPProbe(u)
	case "dns":
		return NewDNSProbe(u)
	case "exec":
		return NewExecuteProbe(u)
	case "source":
		return NewSourceProbe(u)
	case "dummy":
		return NewDummyProbe(u)
	default:
		return NewPluginProbe(u)
	}
}

func New(rawURL string) (Probe, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, ErrInvalidURL
	}

	if u.Scheme == "" {
		return nil, ErrMissingScheme
	}

	return NewFromURL(u)
}

func WithoutPlugin(p Probe, err error) (Probe, error) {
	if err != nil {
		return nil, err
	}

	if _, ok := p.(PluginProbe); ok {
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
