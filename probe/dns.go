package probe

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrUnsupportedDNSType = errors.New("unsupported DNS type")
)

func dnsResolveAuto(ctx context.Context, target string) (string, error) {
	addrs, err := net.DefaultResolver.LookupHost(ctx, target)
	return strings.Join(addrs, "\n"), err
}

func dnsResolveIP(ctx context.Context, protocol, target string) (string, error) {
	ips, err := net.DefaultResolver.LookupIP(ctx, protocol, target)
	addrs := make([]string, len(ips))
	for i, x := range ips {
		addrs[i] = x.String()
	}
	return strings.Join(addrs, "\n"), err
}

func dnsResolveA(ctx context.Context, target string) (string, error) {
	return dnsResolveIP(ctx, "ip4", target)
}

func dnsResolveAAAA(ctx context.Context, target string) (string, error) {
	return dnsResolveIP(ctx, "ip6", target)
}

func dnsResolveCNAME(ctx context.Context, target string) (string, error) {
	return net.DefaultResolver.LookupCNAME(ctx, target)
}

func dnsResolveMX(ctx context.Context, target string) (string, error) {
	mxs, err := net.DefaultResolver.LookupMX(ctx, target)
	addrs := make([]string, len(mxs))
	for i, x := range mxs {
		addrs[i] = x.Host
	}
	return strings.Join(addrs, "\n"), err
}

func dnsResolveNS(ctx context.Context, target string) (string, error) {
	nss, err := net.DefaultResolver.LookupNS(ctx, target)
	addrs := make([]string, len(nss))
	for i, x := range nss {
		addrs[i] = x.Host
	}
	return strings.Join(addrs, "\n"), err
}

func dnsResolveTXT(ctx context.Context, target string) (string, error) {
	texts, err := net.DefaultResolver.LookupTXT(ctx, target)
	return strings.Join(texts, "\n"), err
}

type DNSProbe struct {
	target  *url.URL
	resolve func(ctx context.Context, target string) (string, error)
}

func NewDNSProbe(u *url.URL) (DNSProbe, error) {
	p := DNSProbe{target: &url.URL{Scheme: "dns", Opaque: u.Opaque, Fragment: u.Fragment}}
	if u.Opaque == "" {
		p.target.Opaque = u.Hostname()
	}
	switch strings.ToUpper(u.Query().Get("type")) {
	case "":
		p.resolve = dnsResolveAuto
	case "A":
		p.target.RawQuery = "type=A"
		p.resolve = dnsResolveA
	case "AAAA":
		p.target.RawQuery = "type=AAAA"
		p.resolve = dnsResolveAAAA
	case "CNAME":
		p.target.RawQuery = "type=CNAME"
		p.resolve = dnsResolveCNAME
	case "MX":
		p.target.RawQuery = "type=MX"
		p.resolve = dnsResolveMX
	case "NS":
		p.target.RawQuery = "type=NS"
		p.resolve = dnsResolveNS
	case "TXT":
		p.target.RawQuery = "type=TXT"
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
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	st := time.Now()
	msg, err := p.resolve(ctx, p.target.Opaque)
	d := time.Now().Sub(st)

	rec := api.Record{
		CheckedAt: st,
		Target:    p.target,
		Status:    api.StatusHealthy,
		Message:   msg,
		Latency:   d,
	}

	if err != nil {
		rec.Status = api.StatusFailure
		rec.Message = err.Error()
	}

	r.Report(timeoutOr(ctx, rec))
}
