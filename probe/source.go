package probe

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type invalidURLs []string

func (es invalidURLs) Error() string {
	var ss []string
	for _, e := range es {
		ss = append(ss, e)
	}
	return "Invalid URL: " + strings.Join(ss, ", ")
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
	if u.Opaque == "" {
		return &url.URL{Scheme: "source", Opaque: u.Path, Fragment: u.Fragment}
	} else {
		return &url.URL{Scheme: "source", Opaque: u.Opaque, Fragment: u.Fragment}
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
	if u.Scheme == "source" {
		return normalizeSourceURL(u), nil
	}
	return u, nil
}

type SourceProbe struct {
	target *url.URL
}

func NewSourceProbe(u *url.URL) (SourceProbe, error) {
	s := SourceProbe{
		target: normalizeSourceURL(u),
	}
	err := s.load(nil, make(map[string]Probe))
	return s, err
}

func (p SourceProbe) Target() *url.URL {
	return p.target
}

func (p SourceProbe) open() (io.ReadCloser, error) {
	return os.Open(p.target.Opaque)
}

func (p SourceProbe) load(ignores ignoreSet, out map[string]Probe) error {
	f, err := p.open()
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
			err := SourceProbe{target}.load(append(ignores, p.target.String()), out)
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
	if err := p.load(nil, probes); err != nil {
		d := time.Now().Sub(stime)
		r.Report(api.Record{
			CheckedAt: stime,
			Target:    p.target,
			Status:    api.StatusUnknown,
			Message:   err.Error(),
			Latency:   d,
		})
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
