package probe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

var (
	HTTPUserAgent = "ayd health check"
)

const (
	HTTP_REDIRECT_MAX = 10
)

var (
	ErrRedirectLoopDetected = errors.New("redirect loop detected")
	httpClient              = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: 10 * time.Minute,
		},
		CheckRedirect: checkHTTPRedirect,
	}
)

func checkHTTPRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > HTTP_REDIRECT_MAX {
		return ErrRedirectLoopDetected
	}
	return nil
}

type HTTPProbe struct {
	target  *url.URL
	client  *http.Client
	request *http.Request
}

func NewHTTPProbe(u *url.URL) (HTTPProbe, error) {
	ucopy := *u
	requrl := &ucopy

	scheme, separator, method := SplitScheme(requrl.Scheme)
	method = strings.ToUpper(method)

	requrl.Scheme = scheme

	if separator == 0 {
		method = "GET"
	} else if separator != '-' {
		return HTTPProbe{}, ErrUnsupportedScheme
	} else {
		switch method {
		case "GET", "HEAD", "POST", "OPTIONS", "CONNECT":
		default:
			return HTTPProbe{}, fmt.Errorf("HTTP \"%s\" method is not supported. Please use GET, HEAD, POST, OPTIONS, or CONNECT.", method)
		}
	}

	return HTTPProbe{
		target: u,
		client: httpClient,
		request: &http.Request{
			Method: method,
			URL:    requrl,
			Header: http.Header{
				"User-Agent": {HTTPUserAgent},
			},
		},
	}, nil
}

func (p HTTPProbe) Target() *url.URL {
	return p.target
}

func (p HTTPProbe) Check(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	req := p.request.WithContext(ctx)

	st := time.Now()
	resp, err := p.client.Do(req)
	d := time.Now().Sub(st)

	status := api.StatusFailure
	message := ""
	if err != nil {
		message = err.Error()

		dnsErr := &net.DNSError{}
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			status = api.StatusUnknown
		}

		opErr := &net.OpError{}
		if errors.As(err, &opErr) && opErr.Op == "dial" {
			message = fmt.Sprintf("%s: connection refused", opErr.Addr)
		}
	} else {
		message = fmt.Sprintf("proto=%s length=%d status=%s", resp.Proto, resp.ContentLength, strings.ReplaceAll(resp.Status, " ", "_"))
		if 200 <= resp.StatusCode && resp.StatusCode <= 299 {
			status = api.StatusHealthy
		}
	}

	r.Report(timeoutOr(ctx, api.Record{
		CheckedAt: st,
		Target:    p.target,
		Status:    status,
		Message:   message,
		Latency:   d,
	}))
}
