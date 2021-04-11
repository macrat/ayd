package probe

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/macrat/ayd/store"
)

const (
	USER_AGENT = "ayd/0.1.0 health check"
)

type HTTPProbe struct {
	method string
	target *url.URL
	requrl *url.URL
	client *http.Client
}

func NewHTTPProbe(u *url.URL) HTTPProbe {
	ucopy := *u
	requrl := &ucopy

	scheme := strings.Split(requrl.Scheme, "-")
	requrl.Scheme = scheme[0]

	method := "GET"
	if len(scheme) > 1 {
		m := strings.ToUpper(scheme[1])
		switch m {
		case "HEAD", "POST", "OPTIONS":
			method = m
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
		},
	}
}

func (p HTTPProbe) Target() *url.URL {
	return p.target
}

func (p HTTPProbe) Check() store.Record {
	req := &http.Request{
		Method: p.method,
		URL:    p.requrl,
		Header: http.Header{
			"User-Agent": {USER_AGENT},
		},
	}

	st := time.Now()
	resp, err := p.client.Do(req)
	d := time.Now().Sub(st)

	status := store.STATUS_FAIL
	message := ""
	if err != nil {
		message = err.Error()
		status = store.STATUS_UNKNOWN
	} else {
		message = resp.Status
		if 200 <= resp.StatusCode && resp.StatusCode <= 299 {
			status = store.STATUS_OK
		}
	}

	return store.Record{
		CheckedAt: st,
		Target:    p.target,
		Status:    status,
		Message:   message,
		Latency:   d,
	}
}
