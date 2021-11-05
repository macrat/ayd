package probe

import (
	"context"
	"errors"
	"net/url"

	"github.com/macrat/ayd/internal/scheme"
	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrInvalidURL        = errors.New("invalid URL")
	ErrMissingScheme     = errors.New("missing scheme in URL")
	ErrUnsupportedScheme = errors.New("unsupported scheme")
	ErrMissingHost       = errors.New("missing target host")
)

// Reporter is a shorthand to ayd/internal/scheme.Reporter.
type Reporter = scheme.Reporter

func SplitScheme(scheme string) (probe string, separator rune, variant string) {
	for i, x := range scheme {
		if x == '-' || x == '+' {
			return scheme[:i], x, scheme[i+1:]
		}
	}
	return scheme, 0, ""
}

type Probe interface {
	Target() *url.URL
	Check(context.Context, Reporter)
}

func NewFromURL(u *url.URL) (Probe, error) {
	scheme, _, _ := SplitScheme(u.Scheme)

	switch scheme {
	case "http", "https":
		return NewHTTPProbe(u)
	case "ping", "ping4", "ping6":
		return NewPingProbe(u)
	case "tcp", "tcp4", "tcp6":
		return NewTCPProbe(u)
	case "dns", "dns4", "dns6":
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
