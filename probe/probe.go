package probe

import (
	"errors"
	"net/url"

	"github.com/macrat/ayd/store"
)

var (
	InvalidURIError        = errors.New("invalid URI")
	MissingSchemeError     = errors.New("missing scheme")
	UnsupportedSchemeError = errors.New("unsupported scheme")
)

type Probe interface {
	Target() *url.URL
	Check() store.Record
}

func GetByURL(u *url.URL) Probe {
	switch u.Scheme {
	case
		"http", "https",
		"http-get", "https-get",
		"http-head", "https-head",
		"http-post", "https-post",
		"http-options", "https-options":
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
		return nil
	}
}

func Get(rawURL string) (Probe, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, InvalidURIError
	}

	if u.Scheme == "" {
		return nil, MissingSchemeError
	}

	p := GetByURL(u)
	if p == nil {
		return nil, UnsupportedSchemeError
	}

	return p, nil
}
