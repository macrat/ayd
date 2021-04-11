package probe

import (
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

type Probe interface {
	Target() *url.URL
	Check() store.Record
}

func GetByURL(u *url.URL) (Probe, error) {
	if strings.HasPrefix(u.Scheme, "http-") || strings.HasPrefix(u.Scheme, "https-") {
		return NewHTTPProbe(u)
	}

	switch u.Scheme {
	case "http", "https":
		return NewHTTPProbe(u)
	case "ping":
		return NewPingProbe(u)
	case "tcp":
		return NewTCPProbe(u)
	case "dns":
		return NewDNSProbe(u)
	case "exec":
		return NewExecuteProbe(u)
	default:
		return nil, ErrUnsupportedScheme
	}
}

func Get(rawURL string) (Probe, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, ErrInvalidURI
	}

	if u.Scheme == "" {
		return nil, ErrMissingScheme
	}

	return GetByURL(u)
}
