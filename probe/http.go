package probe

import (
	"net/http"
	"net/url"
	"time"
)

const (
	USER_AGENT = "ayd/0.1.0 health check"
)

func HTTPProbe(u *url.URL) Result {
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: 10 * time.Minute,
		},
	}

	req := &http.Request{
		Method: "HEAD",
		URL:    u,
		Header: http.Header{
			"User-Agent": {USER_AGENT},
		},
	}

	st := time.Now()
	resp, err := client.Do(req)
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
		Target:    u,
		Status:    status,
		Message:   message,
		Latency:   d,
	}
}
