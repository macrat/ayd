package probe

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/macrat/ayd/store"
)

var (
	ErrInvalidURI        = errors.New("invalid URI")
	ErrMissingScheme     = errors.New("missing scheme in URI")
	ErrUnsupportedScheme = errors.New("unsupported scheme")
)

type Reporter interface {
	Report(r store.Record)
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
		return nil, ErrInvalidURI
	}

	if u.Scheme == "" {
		return nil, ErrMissingScheme
	}

	return NewFromURL(u)
}

func timeoutOr(ctx context.Context, r store.Record) store.Record {
	switch ctx.Err() {
	case context.Canceled:
		r.Status = store.STATUS_ABORTED
		r.Message = "probe aborted"
	case context.DeadlineExceeded:
		r.Status = store.STATUS_UNKNOWN
		r.Message = "probe timed out"
	default:
	}
	return r
}
