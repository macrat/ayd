package scheme

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

// dnsResolver is a DNS resolver for DNSProbe.
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

type dnsResolveFunc func(ctx context.Context, target string) (string, error)

// getFunc returns a function to resolve DNS for given DNS type.
func (r dnsResolver) getFunc(typ string) (fn dnsResolveFunc, err error) {
	switch strings.ToUpper(typ) {
	case "":
		return r.auto, nil
	case "A":
		return r.a, nil
	case "AAAA":
		return r.aaaa, nil
	case "CNAME":
		return r.cname, nil
	case "MX":
		return r.mx, nil
	case "NS":
		return r.ns, nil
	case "TXT":
		return r.txt, nil
	default:
		return nil, ErrUnsupportedDNSType
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

// DNSProbe is a Prober implementation for the DNS protocol.
type DNSProbe struct {
	target     *api.URL
	targetName string
	resolve    dnsResolveFunc
}

// getDNSTypeByScheme gets DNS Type from URL scheme such as dns-txt or dns4.
func getDNSTypeByScheme(fullScheme string) (typ string, err error) {
	scheme, separator, variant := SplitScheme(fullScheme)

	switch {
	case scheme == "dns" && separator == 0:
		return "", nil
	case scheme == "dns" && separator == '-' && variant != "":
		return strings.ToUpper(variant), nil
	case scheme == "dns4":
		return "A", nil
	case scheme == "dns6":
		return "AAAA", nil
	default:
		return "", ErrUnsupportedScheme
	}
}

// getDNSTypeQuery gets DNS Type from URL query.
// Please use it instead of URL.Query().Get("type") because it should case-insensitive.
func getDNSTypeByQuery(query url.Values) string {
	for k, v := range query {
		if strings.ToUpper(k) == "TYPE" {
			return strings.ToUpper(v[len(v)-1])
		}
	}
	return ""
}

// NewDNSProbe creates a new DNSProbe.
func NewDNSProbe(u *api.URL) (DNSProbe, error) {
	s := DNSProbe{
		target: &api.URL{
			Scheme:   "dns",
			Opaque:   strings.ToLower(u.Opaque),
			Fragment: u.Fragment,
		},
		targetName: u.Opaque,
	}
	if u.Opaque == "" {
		s.target.Host = strings.ToLower(u.Host)
		s.targetName = strings.ToLower(strings.SplitN(strings.TrimLeft(u.Path, "/"), "/", 2)[0])
		s.target.Path = "/" + s.targetName

		if s.target.Host == "" {
			s.target.Opaque = s.targetName
			s.target.Path = ""
		}
	}
	if s.targetName == "" {
		return DNSProbe{}, ErrMissingDomainName
	}

	typ := getDNSTypeByQuery(u.ToURL().Query())

	if t, err := getDNSTypeByScheme(u.Scheme); err != nil {
		return DNSProbe{}, err
	} else if t != "" {
		if typ != "" && typ != t {
			return DNSProbe{}, ErrConflictDNSType
		}
		u.RawQuery = "type=" + t
		typ = t
	}

	var err error
	s.resolve, err = newDNSResolver(s.target.Host).getFunc(typ)
	if err != nil {
		return DNSProbe{}, err
	}

	if typ != "" {
		s.target.RawQuery = "type=" + typ
	}

	return s, nil
}

func (s DNSProbe) Target() *api.URL {
	return s.target
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

func (s DNSProbe) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	st := time.Now()
	msg, err := s.resolve(ctx, s.targetName)
	d := time.Since(st)

	rec := api.Record{
		Time:    st,
		Target:  s.target,
		Status:  api.StatusHealthy,
		Message: msg,
		Latency: d,
	}

	if err != nil {
		rec.Status = api.StatusFailure
		rec.Message = err.Error()

		dnsErr := &net.DNSError{}
		if errors.As(err, &dnsErr) {
			if s.target.Host != "" {
				dnsErr.Server = s.target.Host
			}
			rec.Message = dnsErrorToMessage(dnsErr)
		}
	}

	r.Report(s.target, timeoutOr(ctx, rec))
}
