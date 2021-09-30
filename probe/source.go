package probe

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrInvalidSource = errors.New("invalid source")
)

type invalidURLs []string

func (es invalidURLs) Error() string {
	var ss []string
	for _, e := range es {
		ss = append(ss, e)
	}
	return "invalid URL: " + strings.Join(ss, ", ")
}

type ignoreSet []string

func (is ignoreSet) Has(s string) bool {
	for _, i := range is {
		if i == s {
			return true
		}
	}
	return false
}

func normalizeSourceURL(u *url.URL) *url.URL {
	switch u.Scheme {
	case "source+http", "source+https":
		return u
	case "source+exec":
		path := u.Opaque
		if u.Opaque == "" {
			path = u.Path
		}
		return &url.URL{
			Scheme:   "source+exec",
			Opaque:   filepath.ToSlash(path),
			RawQuery: u.RawQuery,
			Fragment: u.Fragment,
		}
	default:
		if u.Opaque == "" {
			return &url.URL{Scheme: "source", Opaque: u.Path, Fragment: u.Fragment}
		} else {
			return &url.URL{Scheme: "source", Opaque: u.Opaque, Fragment: u.Fragment}
		}
	}
}

type sourceScanner struct {
	Scanner *bufio.Scanner
	Text    string
}

func (s *sourceScanner) Scan() bool {
	for {
		if !s.Scanner.Scan() {
			return false
		}
		s.Text = strings.TrimSpace(s.Scanner.Text())

		if s.Text != "" && !strings.HasPrefix(s.Text, "#") {
			return true
		}
	}
}

func (s *sourceScanner) URL() (*url.URL, error) {
	u, err := url.Parse(s.Text)
	if err != nil {
		return nil, err
	}

	scheme := strings.SplitN(u.Scheme, "-", 2)[0]
	scheme = strings.SplitN(scheme, "+", 2)[0]
	if scheme == "source" {
		return normalizeSourceURL(u), nil
	}

	return u, nil
}

type SourceProbe struct {
	target *url.URL
}

func NewSourceProbe(u *url.URL) (SourceProbe, error) {
	if strings.Contains(u.Scheme, "-") {
		return SourceProbe{}, ErrUnsupportedScheme
	}
	scheme := strings.SplitN(u.Scheme, "+", 2)
	if len(scheme) > 1 {
		switch scheme[1] {
		case "http", "https", "exec", "":
			break
		default:
			return SourceProbe{}, ErrUnsupportedScheme
		}
	}

	s := SourceProbe{
		target: normalizeSourceURL(u),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err := s.load(ctx, nil, make(map[string]Probe))
	if err != nil {
		err = fmt.Errorf("%w: %s", ErrInvalidSource, err)
	}

	return s, err
}

func (p SourceProbe) Target() *url.URL {
	return p.target
}

func (p SourceProbe) open(ctx context.Context) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	switch p.target.Scheme {
	case "source+http", "source+https":
		u := *p.target
		u.Scheme = u.Scheme[len("source+"):]
		resp, err := httpClient.Do((&http.Request{
			Method: "GET",
			URL:    &u,
			Header: http.Header{
				"User-Agent": {HTTPUserAgent},
			},
		}).WithContext(ctx))
		switch {
		case err != nil:
			return nil, err
		case resp.StatusCode != http.StatusOK:
			return nil, fmt.Errorf("%w: failed to fetch", ErrInvalidURL)
		default:
			return resp.Body, nil
		}
	case "source+exec":
		var args []string
		if p.target.Fragment != "" {
			args = []string{p.target.Fragment}
		}

		cmd := exec.CommandContext(ctx, filepath.FromSlash(p.target.Opaque), args...)
		cmd.Env = getExecuteEnvByURL(p.target)

		stdout := &bytes.Buffer{}
		cmd.Stdout = stdout
		stderr := &bytes.Buffer{}
		cmd.Stderr = stderr

		err := cmd.Run()
		if err != nil {
			return nil, fmt.Errorf("%s: failed to execute: %s", ErrInvalidURL, err)
		}

		if stderr.Len() != 0 {
			return nil, fmt.Errorf("%w: failed to execute: %s", ErrInvalidURL, stderr.String())
		}

		return io.NopCloser(bytes.NewReader(stdout.Bytes())), nil
	default:
		return os.Open(p.target.Opaque)
	}
}

func (p SourceProbe) load(ctx context.Context, ignores ignoreSet, out map[string]Probe) error {
	f, err := p.open(ctx)
	if err != nil {
		return err
	}
	defer f.Close()

	var invalids invalidURLs

	scanner := &sourceScanner{Scanner: bufio.NewScanner(f)}
	for scanner.Scan() {
		target, err := scanner.URL()
		if err != nil {
			invalids = append(invalids, scanner.Text)
			continue
		}

		if target.Scheme != "source" {
			probe, err := NewFromURL(target)
			if err != nil {
				invalids = append(invalids, scanner.Text)
			} else {
				out[probe.Target().String()] = probe
			}
		} else if !ignores.Has(target.String()) {
			err := SourceProbe{target}.load(ctx, append(ignores, p.target.String()), out)
			if es, ok := err.(invalidURLs); ok {
				invalids = append(invalids, es...)
			} else if err != nil {
				invalids = append(invalids, scanner.Text)
			}
		}
	}

	if len(invalids) > 0 {
		return invalids
	}

	return nil
}

func (p SourceProbe) Check(ctx context.Context, r Reporter) {
	stime := time.Now()

	probes := make(map[string]Probe)
	if err := p.load(ctx, nil, probes); err != nil {
		d := time.Now().Sub(stime)
		r.Report(timeoutOr(ctx, api.Record{
			CheckedAt: stime,
			Target:    p.target,
			Status:    api.StatusUnknown,
			Message:   err.Error(),
			Latency:   d,
		}))
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}

	for _, p := range probes {
		wg.Add(1)

		go func(p Probe) {
			p.Check(ctx, r)
			wg.Done()
		}(p)
	}
	wg.Wait()

	d := time.Now().Sub(stime)
	r.Report(api.Record{
		CheckedAt: stime,
		Target:    p.target,
		Status:    api.StatusHealthy,
		Message:   fmt.Sprintf("target_count=%d", len(probes)),
		Latency:   d,
	})
}
