package probe

import (
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/macrat/ayd/store"
)

type DNSProbe struct {
	target *url.URL
}

func NewDNSProbe(u *url.URL) (DNSProbe, error) {
	if u.Opaque != "" {
		return DNSProbe{&url.URL{Scheme: "dns", Opaque: u.Opaque}}, nil
	} else {
		return DNSProbe{&url.URL{Scheme: "dns", Opaque: u.Hostname()}}, nil
	}
}

func (p DNSProbe) Target() *url.URL {
	return p.target
}

func (p DNSProbe) Check() []store.Record {
	st := time.Now()
	addrs, err := net.LookupHost(p.target.Opaque)
	d := time.Now().Sub(st)

	r := store.Record{
		CheckedAt: st,
		Target:    p.target,
		Latency:   d,
	}

	if err != nil {
		r.Status = store.STATUS_FAILURE
		r.Message = err.Error()
	} else {
		r.Status = store.STATUS_HEALTHY
		r.Message = strings.Join(addrs, ", ")
	}

	return []store.Record{r}
}
