package scheme

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
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

type HTTPScheme struct {
	target  *url.URL
	request *http.Request
}

func NewHTTPScheme(u *url.URL) (HTTPScheme, error) {
	u.Host = strings.ToLower(u.Host)

	ucopy := *u
	requrl := &ucopy

	scheme, separator, method := SplitScheme(requrl.Scheme)
	method = strings.ToUpper(method)

	requrl.Scheme = scheme

	if separator == 0 {
		method = "GET"
	} else if separator != '-' {
		return HTTPScheme{}, ErrUnsupportedScheme
	} else {
		switch method {
		case "GET", "HEAD", "POST", "OPTIONS", "CONNECT":
		default:
			return HTTPScheme{}, fmt.Errorf("HTTP \"%s\" method is not supported. Please use GET, HEAD, POST, OPTIONS, or CONNECT.", method)
		}
	}

	if u.Hostname() == "" {
		return HTTPScheme{}, ErrMissingHost
	}

	if u.Path == "" {
		u.Path = "/"
	}

	return HTTPScheme{
		target: u,
		request: &http.Request{
			Method: method,
			URL:    requrl,
			Header: http.Header{
				"User-Agent": {HTTPUserAgent},
			},
		},
	}, nil
}

func (s HTTPScheme) Target() *url.URL {
	return s.target
}

func (s HTTPScheme) responseToRecord(resp *http.Response, err error) api.Record {
	status := api.StatusFailure
	message := ""

	if err == nil {
		message = fmt.Sprintf("proto=%s length=%d status=%s", resp.Proto, resp.ContentLength, strings.ReplaceAll(resp.Status, " ", "_"))
		if 200 <= resp.StatusCode && resp.StatusCode <= 299 {
			status = api.StatusHealthy
		}
	} else {
		message = err.Error()

		dnsErr := &net.DNSError{}
		opErr := &net.OpError{}

		if errors.As(err, &dnsErr) {
			status = api.StatusUnknown
			message = dnsErrorToMessage(dnsErr)
		} else if errors.As(err, &opErr) && opErr.Op == "dial" {
			message = fmt.Sprintf("%s: connection refused", opErr.Addr)
		}
	}

	return api.Record{
		Target:  s.target,
		Status:  status,
		Message: message,
	}
}

func (s HTTPScheme) run(ctx context.Context, r Reporter, req *http.Request) {
	st := time.Now()
	resp, err := httpClient.Do(req)
	d := time.Since(st)

	rec := s.responseToRecord(resp, err)
	rec.CheckedAt = st
	rec.Latency = d

	r.Report(s.target, timeoutOr(ctx, rec))
}

func (s HTTPScheme) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	req := s.request.Clone(ctx)

	s.run(ctx, r, req)
}

func (s HTTPScheme) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	qs := s.target.Query()
	qs.Set("ayd_checked_at", lastRecord.CheckedAt.Format(time.RFC3339))
	qs.Set("ayd_status", lastRecord.Status.String())
	qs.Set("ayd_latency", strconv.FormatFloat(float64(lastRecord.Latency.Microseconds())/1000.0, 'f', -1, 64))
	qs.Set("ayd_target", lastRecord.Target.String())
	qs.Set("ayd_message", lastRecord.Message)

	u := *s.target
	u.RawQuery = qs.Encode()

	req := s.request.Clone(ctx)
	req.URL = &u

	s.run(ctx, AlertReporter{s.target, r}, req)
}
