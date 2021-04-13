package probe

import (
	"bufio"
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

type SourceProbe struct {
	target *url.URL
}

func NewSourceProbe(u *url.URL) (SourceProbe, error) {
	s := SourceProbe{}
	if u.Opaque == "" {
		s.target = &url.URL{Scheme: "source", Opaque: u.Path}
	} else {
		s.target = &url.URL{Scheme: "source", Opaque: u.Opaque}
	}
	_, err := s.load()
	return s, err
}

func (p SourceProbe) Target() *url.URL {
	return p.target
}

func (p SourceProbe) load() ([]Probe, error) {
	f, err := os.Open(p.target.Opaque)
	if err != nil {
		return nil, err
	}

	var probes []Probe
	var invalids invalidURIs

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		target := strings.TrimSpace(scanner.Text())

		if target == "" || strings.HasPrefix(target, "#") {
			continue
		}

		probe, err := Get(target)
		if err != nil {
			invalids = append(invalids, target)
		} else {
			probes = append(probes, probe)
		}
	}

	if len(invalids) > 0 {
		return nil, invalids
	}

	return probes, nil
}

func (p SourceProbe) Check() []store.Record {
	stime := time.Now()

	probes, err := p.load()
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

	ch := make(chan []store.Record, len(probes))
	wg := &sync.WaitGroup{}

	for _, p := range probes {
		wg.Add(1)

		go func(p Probe, ch chan []store.Record) {
			ch <- p.Check()
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
