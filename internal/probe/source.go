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

	"github.com/macrat/ayd/internal/ayderr"
	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrInvalidSource = errors.New("invalid source")
	ErrMissingFile   = errors.New("missing file")
)

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

	scheme, _, _ := SplitScheme(u.Scheme)
	if scheme == "source" {
		return normalizeSourceURL(u), nil
	}

	return u, nil
}

type SourceProbe struct {
	target *url.URL
}

func NewSourceProbe(u *url.URL) (SourceProbe, error) {
	_, separator, variant := SplitScheme(u.Scheme)

	if separator == '-' {
		return SourceProbe{}, ErrUnsupportedScheme
	}

	switch variant {
	case "":
		if u.Opaque == "" && u.Path == "" {
			return SourceProbe{}, ErrMissingFile
		}
	case "http", "https":
		if u.Hostname() == "" {
			return SourceProbe{}, ErrMissingHost
		}
	case "exec":
		if u.Opaque == "" && u.Path == "" {
			return SourceProbe{}, ErrMissingCommand
		}
	default:
		return SourceProbe{}, ErrUnsupportedScheme
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
		}).Clone(ctx))
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
			return nil, fmt.Errorf("%w: failed to execute: %s", ErrInvalidURL, err)
		}

		if stderr.Len() != 0 {
			return nil, fmt.Errorf("%w: failed to execute: %s", ErrInvalidURL, autoDecode(stderr.Bytes()))
		}

		return io.NopCloser(strings.NewReader(autoDecode(stdout.Bytes()))), nil
	default:
		f, err := os.Open(p.target.Opaque)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		// XXX: can I make io.Reader instead of read all at here?
		bs, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}

		return io.NopCloser(strings.NewReader(autoDecode(bs))), nil
	}
}

func (p SourceProbe) load(ctx context.Context, ignores ignoreSet, out map[string]Probe) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	f, err := p.open(ctx)
	if err != nil {
		return err
	}
	defer f.Close()

	invalids := &ayderr.ListBuilder{What: ErrInvalidURL}

	scanner := &sourceScanner{Scanner: bufio.NewScanner(f)}
	for scanner.Scan() {
		target, err := scanner.URL()
		if err != nil {
			invalids.Pushf("%s", scanner.Text)
			continue
		}

		if target.Scheme != "source" {
			probe, err := NewFromURL(target)
			if err != nil {
				invalids.Pushf("%s", target)
			} else {
				out[probe.Target().String()] = probe
			}
		} else if !ignores.Has(target.String()) {
			err := SourceProbe{target}.load(ctx, append(ignores, p.target.String()), out)
			es := ayderr.List{}
			if errors.As(err, &es) {
				invalids.Push(es.Children...)
			} else if err != nil {
				invalids.Pushf("%s", target)
			}
		}

		select {
		case <-ctx.Done():
			return errors.New("context cancelled")
		default:
		}
	}

	return invalids.Build()
}

func (p SourceProbe) Check(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stime := time.Now()

	probes := make(map[string]Probe)
	if err := p.load(ctx, nil, probes); err != nil {
		d := time.Now().Sub(stime)
		r.Report(timeoutOr(ctx, api.Record{
			CheckedAt: stime,
			Target:    p.target,
			Status:    api.StatusFailure,
			Message:   err.Error(),
			Latency:   d,
		}))
		return
	}

	d := time.Now().Sub(stime)
	r.Report(api.Record{
		CheckedAt: stime,
		Target:    p.target,
		Status:    api.StatusHealthy,
		Message:   fmt.Sprintf("target_count=%d", len(probes)),
		Latency:   d,
	})

	wg := &sync.WaitGroup{}

	for _, p := range probes {
		wg.Add(1)

		go func(p Probe) {
			p.Check(ctx, r)
			wg.Done()
		}(p)
	}
	wg.Wait()
}
