package probe

import (
	"fmt"
	"net/url"

	"github.com/macrat/ayd/store"
)

type Probe interface {
	Target() *url.URL
	Check() store.Record
}

func GetByURL(u *url.URL) Probe {
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
		return nil
	}
}

func Get(rawURL string) (Probe, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target: %s", rawURL)
	}

	if u.Scheme == "" {
		u, err = url.Parse("ping:" + rawURL)
		if err != nil {
			return nil, fmt.Errorf("invalid target: %s", rawURL)
		}
	}

	p := GetByURL(u)
	if p == nil {
		return nil, fmt.Errorf("unsupported scheme: %#v", u.Scheme)
	}

	return p, nil
}
