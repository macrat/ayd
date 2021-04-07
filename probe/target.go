package probe

import (
	"fmt"
	"net/url"
)

func ParseTarget(target string) (*url.URL, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid target: %s", target)
	}

	if u.Scheme == "" {
		u, err = url.Parse("ping:" + target)
		if err != nil {
			return nil, fmt.Errorf("invalid target: %s", target)
		}
	}

	switch u.Scheme {
	case "http", "https":
		return u, nil
	case "ping":
		if u.Opaque != "" {
			return &url.URL{Scheme: "ping", Opaque: u.Opaque}, nil
		} else {
			return &url.URL{Scheme: "ping", Opaque: u.Hostname()}, nil
		}
	case "tcp":
		if u.Opaque != "" {
			return &url.URL{Scheme: "tcp", Opaque: u.Opaque}, nil
		} else {
			return &url.URL{Scheme: "tcp", Opaque: u.Host}, nil
		}
	case "dns":
		if u.Opaque != "" {
			return &url.URL{Scheme: "dns", Opaque: u.Opaque}, nil
		} else {
			return &url.URL{Scheme: "dns", Opaque: u.Hostname()}, nil
		}
	case "exec":
		path := u.Opaque
		if u.Opaque == "" {
			path = u.Path
		}
		return &url.URL{
			Scheme:   "exec",
			Path:     path,
			RawQuery: u.RawQuery,
			Fragment: u.Fragment,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", target)
	}
}
