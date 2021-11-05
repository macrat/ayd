package scheme

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

func normalizeSourceURL(u *url.URL) (*url.URL, error) {
	switch u.Scheme {
	case "source+http", "source+https":
		if u.Hostname() == "" {
			return nil, ErrMissingHost
		}
		return u, nil
	case "source+exec":
		path := u.Opaque
		if u.Opaque == "" {
			path = u.Path

			if path == "" {
				return nil, ErrMissingCommand
			}
		}
		return &url.URL{
			Scheme:   "source+exec",
			Opaque:   filepath.ToSlash(path),
			RawQuery: u.RawQuery,
			Fragment: u.Fragment,
		}, nil
	case "source":
		path := u.Opaque
		if u.Opaque == "" {
			path = u.Path

			if path == "" {
				return nil, ErrMissingFile
			}
		}
		return &url.URL{
			Scheme:   "source",
			Opaque:   filepath.ToSlash(path),
			Fragment: u.Fragment,
		}, nil
	default:
		return nil, ErrUnsupportedScheme
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

	if s, _, _ := SplitScheme(u.Scheme); s == "source" {
		return normalizeSourceURL(u)
	}

	return u, nil
}

type SourceProbe struct {
	target  *url.URL
	tracker *TargetTracker
}

func NewSourceProbe(u *url.URL) (SourceProbe, error) {
	var err error
	u, err = normalizeSourceURL(u)
	if err != nil {
		return SourceProbe{}, err
	}

	s := SourceProbe{
		target:  u,
		tracker: &TargetTracker{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err = s.load(ctx, nil, make(map[string]Prober))
	if err != nil {
		return SourceProbe{}, fmt.Errorf("%w: %s", ErrInvalidSource, err)
	}

	return s, nil
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

func (p SourceProbe) load(ctx context.Context, ignores ignoreSet, out map[string]Prober) error {
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
			prober, err := NewProberFromURL(target)
			if err != nil {
				invalids.Pushf("%s", target)
			} else {
				out[prober.Target().String()] = prober
			}
		} else if !ignores.Has(target.String()) {
			err := SourceProbe{target: target}.load(ctx, append(ignores, p.target.String()), out)
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

func (p SourceProbe) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stime := time.Now()

	probes := make(map[string]Prober)
	if err := p.load(ctx, nil, probes); err != nil {
		d := time.Now().Sub(stime)
		r.Report(p.target, timeoutOr(ctx, api.Record{
			CheckedAt: stime,
			Target:    p.target,
			Status:    api.StatusFailure,
			Message:   err.Error(),
			Latency:   d,
		}))
		return
	}

	d := time.Now().Sub(stime)
	r.Report(p.target, api.Record{
		CheckedAt: stime,
		Target:    p.target,
		Status:    api.StatusHealthy,
		Message:   fmt.Sprintf("target_count=%d", len(probes)),
		Latency:   d,
	})

	r = p.tracker.PrepareReporter(p.target, r)

	wg := &sync.WaitGroup{}
	for _, p := range probes {
		wg.Add(1)

		go func(p Prober) {
			p.Probe(ctx, r)
			wg.Done()
		}(p)
	}
	wg.Wait()

	r.DeactivateTarget(p.target, p.tracker.Inactives()...)
}
