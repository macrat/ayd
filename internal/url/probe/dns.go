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
	ErrConflictDNSType    = errors.New("DNS type in scheme and query is conflicted")
	ErrMissingDomainName  = errors.New("missing domain name")
)

type dnsResolver struct {
	Resolver *net.Resolver
}

func newDNSResolver(server string) dnsResolver {
	if server == "" {
		return dnsResolver{net.DefaultResolver}
	} else {
		_, _, err := net.SplitHostPort(server)
		if err != nil {
			server += ":53"
		}
		return dnsResolver{&net.Resolver{
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, server)
			},
		}}
	}
}

func (r dnsResolver) auto(ctx context.Context, target string) (string, error) {
	addrs, err := r.Resolver.LookupHost(ctx, target)
	return "ip=" + strings.Join(addrs, ","), err
}

func (r dnsResolver) ip(ctx context.Context, protocol, target string) (string, error) {
	ips, err := r.Resolver.LookupIP(ctx, protocol, target)
	addrs := make([]string, len(ips))
	for i, x := range ips {
		addrs[i] = x.String()
	}
	return "ip=" + strings.Join(addrs, ","), err
}

func (r dnsResolver) a(ctx context.Context, target string) (string, error) {
	return r.ip(ctx, "ip4", target)
}

func (r dnsResolver) aaaa(ctx context.Context, target string) (string, error) {
	return r.ip(ctx, "ip6", target)
}

func (r dnsResolver) cname(ctx context.Context, target string) (string, error) {
	host, err := r.Resolver.LookupCNAME(ctx, target)
	return "hostname=" + host, err
}

func (r dnsResolver) mx(ctx context.Context, target string) (string, error) {
	mxs, err := r.Resolver.LookupMX(ctx, target)
	addrs := make([]string, len(mxs))
	for i, x := range mxs {
		addrs[i] = x.Host
	}
	return "mx=" + strings.Join(addrs, ","), err
}

func (r dnsResolver) ns(ctx context.Context, target string) (string, error) {
	nss, err := r.Resolver.LookupNS(ctx, target)
	addrs := make([]string, len(nss))
	for i, x := range nss {
		addrs[i] = x.Host
	}
	return "ns=" + strings.Join(addrs, ","), err
}

func (r dnsResolver) txt(ctx context.Context, target string) (string, error) {
	texts, err := r.Resolver.LookupTXT(ctx, target)
	return strings.Join(texts, "\n"), err
}

type DNSProbe struct {
	target   *url.URL
	hostname string
	resolve  func(ctx context.Context, target string) (string, error)
}

func NewDNSProbe(u *url.URL) (DNSProbe, error) {
	p := DNSProbe{
		target: &url.URL{
			Scheme:   "dns",
			Opaque:   u.Opaque,
			Fragment: u.Fragment,
		},
		hostname: u.Opaque,
	}
	if u.Opaque == "" {
		p.target.Host = u.Host
		p.hostname = strings.SplitN(strings.TrimLeft(u.Path, "/"), "/", 2)[0]
		p.target.Path = "/" + p.hostname

		if p.target.Host == "" {
			p.target.Opaque = p.hostname
			p.target.Path = ""
		}
	}

	if p.hostname == "" {
		return DNSProbe{}, ErrMissingDomainName
	}

	scheme, separator, variant := SplitScheme(u.Scheme)
	shorthand := ""

	switch {
	case scheme == "dns" && separator == 0:
		// do nothing
	case scheme == "dns" && separator == '-' && variant != "":
		shorthand = strings.ToUpper(variant)
	case scheme == "dns4":
		shorthand = "A"
	case scheme == "dns6":
		shorthand = "AAAA"
	default:
		return DNSProbe{}, ErrUnsupportedScheme
	}

	if shorthand != "" {
		q := u.Query().Get("type")
		if q != "" && shorthand != strings.ToUpper(q) {
			return DNSProbe{}, ErrConflictDNSType
		}
		u.RawQuery = "type=" + shorthand
	}

	resolve := newDNSResolver(p.target.Host)

	switch strings.ToUpper(u.Query().Get("type")) {
	case "":
		p.resolve = resolve.auto
	case "A":
		p.target.RawQuery = "type=A"
		p.resolve = resolve.a
	case "AAAA":
		p.target.RawQuery = "type=AAAA"
		p.resolve = resolve.aaaa
	case "CNAME":
		p.target.RawQuery = "type=CNAME"
		p.resolve = resolve.cname
	case "MX":
		p.target.RawQuery = "type=MX"
		p.resolve = resolve.mx
	case "NS":
		p.target.RawQuery = "type=NS"
		p.resolve = resolve.ns
	case "TXT":
		p.target.RawQuery = "type=TXT"
		p.resolve = resolve.txt
	default:
		return DNSProbe{}, ErrUnsupportedDNSType
	}
	return p, nil
}

func (p DNSProbe) Target() *url.URL {
	return p.target
}

func dnsErrorToMessage(err *net.DNSError) string {
	msg := err.Error()
	if err.IsNotFound {
		msg = "lookup " + err.Name + ": not found"
	}
	if err.Server != "" {
		msg += " on " + err.Server
	}
	return msg
}

func (p DNSProbe) Check(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	st := time.Now()
	msg, err := p.resolve(ctx, p.hostname)
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

		dnsErr := &net.DNSError{}
		if errors.As(err, &dnsErr) {
			if p.target.Host != "" {
				dnsErr.Server = p.target.Host
			}
			rec.Message = dnsErrorToMessage(dnsErr)
		}
	}

	r.Report(p.target, timeoutOr(ctx, rec))
}