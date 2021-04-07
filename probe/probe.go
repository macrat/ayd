package probe

import (
	"net/url"
)

type Func func(u *url.URL) Result

func Get(u *url.URL) Func {
	switch u.Scheme {
	case "http", "https":
		return HTTPProbe
	case "ping":
		return PingProbe
	case "tcp":
		return TCPProbe
	case "dns":
		return DNSProbe
	case "exec":
		return ExecuteProbe
	default:
		return nil
	}
}
