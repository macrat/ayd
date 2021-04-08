package probe

import (
	"net/http"
	"net/url"
	"time"
)

const (
	USER_AGENT = "ayd/0.1.0 health check"
)

type HTTPProbe struct {
	target *url.URL
	client *http.Client
}

func NewHTTPProbe(u *url.URL) HTTPProbe {
	return HTTPProbe{
		target: u,
		client: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives:     true,
				ResponseHeaderTimeout: 10 * time.Minute,
			},
		},
	}
}

func (p HTTPProbe) Target() *url.URL {
	return p.target
}

func (p HTTPProbe) Check() Result {
	req := &http.Request{
		Method: "HEAD",
		URL:    p.target,
		Header: http.Header{
			"User-Agent": {USER_AGENT},
		},
	}

	st := time.Now()
	resp, err := p.client.Do(req)
	d := time.Now().Sub(st)

	status := STATUS_FAIL
	message := ""
	if err != nil {
		message = err.Error()
		status = STATUS_UNKNOWN
	} else {
		message = resp.Status
		if 200 <= resp.StatusCode && resp.StatusCode <= 299 {
			status = STATUS_OK
		}
	}

	return Result{
		CheckedAt: st,
		Target:    p.target,
		Status:    status,
		Message:   message,
		Latency:   d,
	}
}
