package probe

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/macrat/ayd/store"
)

var (
	ErrUnsupportedDNSType = errors.New("unsupported DNS type")
)

func dnsResolveAuto(ctx context.Context, r *net.Resolver, target string) (string, error) {
	addrs, err := r.LookupHost(ctx, target)
	return strings.Join(addrs, "\n"), err
}

func dnsResolveIP(ctx context.Context, r *net.Resolver, protocol, target string) (string, error) {
	ips, err := r.LookupIP(ctx, protocol, target)
	addrs := make([]string, len(ips))
	for i, x := range ips {
		addrs[i] = x.String()
	}
	return strings.Join(addrs, "\n"), err
}

func dnsResolveA(ctx context.Context, r *net.Resolver, target string) (string, error) {
	return dnsResolveIP(ctx, r, "ip4", target)
}

func dnsResolveAAAA(ctx context.Context, r *net.Resolver, target string) (string, error) {
	return dnsResolveIP(ctx, r, "ip6", target)
}

func dnsResolveCNAME(ctx context.Context, r *net.Resolver, target string) (string, error) {
	return r.LookupCNAME(ctx, target)
}

func dnsResolveMX(ctx context.Context, r *net.Resolver, target string) (string, error) {
	mxs, err := r.LookupMX(ctx, target)
	addrs := make([]string, len(mxs))
	for i, x := range mxs {
		addrs[i] = x.Host
	}
	return strings.Join(addrs, "\n"), err
}

func dnsResolveNS(ctx context.Context, r *net.Resolver, target string) (string, error) {
	nss, err := r.LookupNS(ctx, target)
	addrs := make([]string, len(nss))
	for i, x := range nss {
		addrs[i] = x.Host
	}
	return strings.Join(addrs, "\n"), err
}

func dnsResolveTXT(ctx context.Context, r *net.Resolver, target string) (string, error) {
	texts, err := r.LookupTXT(ctx, target)
	return strings.Join(texts, "\n"), err
}

type DNSProbe struct {
	target  *url.URL
	resolve func(ctx context.Context, r *net.Resolver, target string) (string, error)
}

func NewDNSProbe(u *url.URL) (DNSProbe, error) {
	p := DNSProbe{target: &url.URL{Scheme: "dns", Opaque: u.Opaque}}
	if u.Opaque == "" {
		p.target.Opaque = u.Hostname()
	}
	switch strings.ToUpper(u.Query().Get("type")) {
	case "":
		p.resolve = dnsResolveAuto
	case "A":
		p.target.RawQuery = url.Values{"type": {"A"}}.Encode()
		p.resolve = dnsResolveA
	case "AAAA":
		p.target.RawQuery = url.Values{"type": {"AAAA"}}.Encode()
		p.resolve = dnsResolveAAAA
	case "CNAME":
		p.target.RawQuery = url.Values{"type": {"CNAME"}}.Encode()
		p.resolve = dnsResolveCNAME
	case "MX":
		p.target.RawQuery = url.Values{"type": {"MX"}}.Encode()
		p.resolve = dnsResolveMX
	case "NS":
		p.target.RawQuery = url.Values{"type": {"NS"}}.Encode()
		p.resolve = dnsResolveNS
	case "TXT":
		p.target.RawQuery = url.Values{"type": {"TXT"}}.Encode()
		p.resolve = dnsResolveTXT
	default:
		return DNSProbe{}, ErrUnsupportedDNSType
	}
	return p, nil
}

func (p DNSProbe) Target() *url.URL {
	return p.target
}

func (p DNSProbe) Check(ctx context.Context, r Reporter) {
	var resolver net.Resolver

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	st := time.Now()
	msg, err := p.resolve(ctx, &resolver, p.target.Opaque)
	d := time.Now().Sub(st)

	rec := store.Record{
		CheckedAt: st,
		Target:    p.target,
		Status:    store.STATUS_HEALTHY,
		Message:   msg,
		Latency:   d,
	}

	if err != nil {
		rec.Status = store.STATUS_FAILURE
		rec.Message = err.Error()
	}

	r.Report(timeoutOr(ctx, rec))
}
