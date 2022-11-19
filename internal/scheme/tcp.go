package scheme

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrTCPPortMissing = errors.New("TCP target's port number is required")
)

// TCPProbe is a Prober implementation for the TCP.
type TCPProbe struct {
	target *api.URL
}

func NewTCPProbe(u *api.URL) (TCPProbe, error) {
	scheme, separator, _ := SplitScheme(u.Scheme)

	if separator != 0 {
		return TCPProbe{}, ErrUnsupportedScheme
	}

	s := TCPProbe{&api.URL{Scheme: scheme, Host: strings.ToLower(u.Host), Fragment: u.Fragment}}
	if u.Host == "" {
		s.target.Host = strings.ToLower(u.Opaque)
	}

	if s.target.ToURL().Hostname() == "" {
		return TCPProbe{}, ErrMissingHost
	}
	if s.target.ToURL().Port() == "" {
		return TCPProbe{}, ErrTCPPortMissing
	}

	return s, nil
}

func (s TCPProbe) Target() *api.URL {
	return s.target
}

func (s TCPProbe) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var dialer net.Dialer

	st := time.Now()
	conn, err := dialer.DialContext(ctx, s.target.Scheme, s.target.Host)
	d := time.Since(st)

	rec := api.Record{
		Time:    st,
		Target:  s.target,
		Latency: d,
	}

	if err != nil {
		rec.Status = api.StatusFailure
		rec.Message = err.Error()

		dnsErr := &net.DNSError{}

		if errors.Is(err, &net.AddrError{}) {
			rec.Status = api.StatusUnknown
		} else if errors.As(err, &dnsErr) {
			rec.Status = api.StatusUnknown
			rec.Message = dnsErrorToMessage(dnsErr)
		}
	} else {
		rec.Status = api.StatusHealthy
		rec.Message = "succeed to connect"
		rec.Extra = map[string]interface{}{
			"source_addr": conn.LocalAddr().String(),
			"target_addr": conn.RemoteAddr().String(),
		}
		conn.Close()
	}

	r.Report(s.target, timeoutOr(ctx, rec))
}
