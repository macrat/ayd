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

// Prober is the interface to check the target is dead or alive.
type Prober interface {
	// Target returns the target URL.
	// This URL should not change during lifetime of the instance.
	Target() *url.URL

	// Probe checks the target is dead or alive, and report result(s) to the Reporter.
	Probe(context.Context, Reporter)
}

func NewProberFromURL(u *url.URL) (Prober, error) {
	scheme, _, _ := SplitScheme(u.Scheme)

	switch scheme {
	case "http", "https":
		return NewHTTPScheme(u)
	case "ftp", "ftps":
		return NewFTPProbe(u)
	case "ping", "ping4", "ping6":
		return NewPingProbe(u)
	case "tcp", "tcp4", "tcp6":
		return NewTCPProbe(u)
	case "dns", "dns4", "dns6":
		return NewDNSProbe(u)
	case "exec":
		return NewExecScheme(u)
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
