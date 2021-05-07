package probe

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type invalidURIs []string

func (es invalidURIs) Error() string {
	var ss []string
	for _, e := range es {
		ss = append(ss, e)
	}
	return "Invalid URI: " + strings.Join(ss, ", ")
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
		return &url.URL{Scheme: "source", Opaque: u.Path}
	} else {
		return &url.URL{Scheme: "source", Opaque: u.Opaque}
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
	_, err := s.load(s.target.Opaque, nil)
	return s, err
}

func (p SourceProbe) Target() *url.URL {
	return p.target
}

func (p SourceProbe) load(path string, ignores ignoreSet) (map[string]Probe, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	probes := make(map[string]Probe)
	var invalids invalidURIs

	scanner := &sourceScanner{Scanner: bufio.NewScanner(f)}
	for scanner.Scan() {
		target, err := scanner.URL()
		if err != nil {
			invalids = append(invalids, scanner.Text)
			continue
		}

		if target.Scheme == "source" {
			if !ignores.Has(target.Opaque) {
				ps, err := p.load(target.Opaque, append(ignores, path))
				if err == nil {
					for k, v := range ps {
						probes[k] = v
					}
				} else if es, ok := err.(invalidURIs); ok {
					invalids = append(invalids, es...)
				} else {
					invalids = append(invalids, scanner.Text)
				}
			}

			continue
		}

		probe, err := NewFromURL(target)
		if err != nil {
			invalids = append(invalids, scanner.Text)
		} else {
			probes[probe.Target().String()] = probe
		}
	}

	if len(invalids) > 0 {
		return nil, invalids
	}

	return probes, nil
}

func (p SourceProbe) Check(ctx context.Context, r Reporter) {
	stime := time.Now()

	probes, err := p.load(p.target.Opaque, nil)
	if err != nil {
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
		Message:   fmt.Sprintf("checked %d targets", len(probes)),
		Latency:   d,
	})
}
