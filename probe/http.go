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

	"github.com/macrat/ayd/store"
)

var (
	HTTPUserAgent = "ayd health check"
)

const (
	HTTP_REDIRECT_MAX = 10
)

var (
	ErrRedirectLoopDetected = errors.New("redirect loop detected")
)

type HTTPProbe struct {
	method string
	target *url.URL
	requrl *url.URL
	client *http.Client
}

func NewHTTPProbe(u *url.URL) (HTTPProbe, error) {
	ucopy := *u
	requrl := &ucopy

	scheme := strings.Split(requrl.Scheme, "-")
	requrl.Scheme = scheme[0]

	var method string
	if len(scheme) > 1 {
		m := strings.ToUpper(scheme[1])
		switch m {
		case "":
			method = "GET"
		case "GET", "HEAD", "POST", "OPTIONS":
			method = m
		default:
			return HTTPProbe{}, fmt.Errorf("HTTP \"%s\" method is not supported. Please use GET, HEAD, POST, or OPTIONS.", m)
		}
	}

	return HTTPProbe{
		method: method,
		target: u,
		requrl: requrl,
		client: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives:     true,
				ResponseHeaderTimeout: 10 * time.Minute,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > HTTP_REDIRECT_MAX {
					return ErrRedirectLoopDetected
				}
				return nil
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

	req := (&http.Request{
		Method: p.method,
		URL:    p.requrl,
		Header: http.Header{
			"User-Agent": {HTTPUserAgent},
		},
	}).WithContext(ctx)

	st := time.Now()
	resp, err := p.client.Do(req)
	d := time.Now().Sub(st)

	status := store.STATUS_FAILURE
	message := ""
	if err != nil {
		message = err.Error()
		if e, ok := errors.Unwrap(errors.Unwrap(err)).(*net.DNSError); ok && e.IsNotFound {
			status = store.STATUS_UNKNOWN
		}
	} else {
		message = resp.Status
		if 200 <= resp.StatusCode && resp.StatusCode <= 299 {
			status = store.STATUS_HEALTHY
		}
	}

	r.Report(timeoutOr(ctx, store.Record{
		CheckedAt: st,
		Target:    p.target,
		Status:    status,
		Message:   message,
		Latency:   d,
	}))
}
