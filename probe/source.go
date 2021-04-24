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

	"github.com/macrat/ayd/store"
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

func (p SourceProbe) load(path string, ignores ignoreSet) ([]Probe, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var probes []Probe
	var invalids invalidURIs

	isDuplicated := func(p Probe) bool {
		for _, x := range probes {
			if x.Target().String() == p.Target().String() {
				return true
			}
		}
		return false
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		target := strings.TrimSpace(scanner.Text())

		if target == "" || strings.HasPrefix(target, "#") {
			continue
		}

		if strings.HasPrefix(target, "source:") {
			u, err := url.Parse(target)
			if err != nil {
				invalids = append(invalids, target)
				continue
			}
			u = normalizeSourceURL(u)
			if ignores.Has(u.Opaque) {
				continue
			}
			ps, err := p.load(u.Opaque, append(ignores, path))
			if err == nil {
				for _, x := range ps {
					if !isDuplicated(x) {
						probes = append(probes, x)
					}
				}
			} else if es, ok := err.(invalidURIs); ok {
				invalids = append(invalids, es...)
			}
			continue
		}

		probe, err := Get(target)
		if err != nil {
			invalids = append(invalids, target)
		} else if !isDuplicated(probe) {
			probes = append(probes, probe)
		}
	}

	if len(invalids) > 0 {
		return nil, invalids
	}

	return probes, nil
}

func (p SourceProbe) Check(ctx context.Context) []store.Record {
	stime := time.Now()

	probes, err := p.load(p.target.Opaque, nil)
	if err != nil {
		d := time.Now().Sub(stime)
		return []store.Record{{
			CheckedAt: stime,
			Target:    p.target,
			Status:    store.STATUS_UNKNOWN,
			Message:   err.Error(),
			Latency:   d,
		}}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan []store.Record, len(probes))
	wg := &sync.WaitGroup{}

	for _, p := range probes {
		wg.Add(1)

		go func(p Probe, ch chan []store.Record) {
			ch <- p.Check(ctx)
			wg.Done()
		}(p, ch)
	}
	wg.Wait()
	close(ch)

	results := []store.Record{}
	for rs := range ch {
		results = append(results, rs...)
	}

	d := time.Now().Sub(stime)
	return append(results, store.Record{
		CheckedAt: stime,
		Target:    p.target,
		Status:    store.STATUS_HEALTHY,
		Message:   fmt.Sprintf("checked %d targets", len(probes)),
		Latency:   d,
	})
}
