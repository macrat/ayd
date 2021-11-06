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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/macrat/ayd/internal/ayderr"
	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrInvalidSource    = errors.New("invalid source")
	ErrInvalidSourceURL = errors.New("invalid source URL")
	ErrMissingFile      = errors.New("missing file")
)

// urlSet is a set of URL.
type urlSet []*url.URL

func (s urlSet) search(u *url.URL) int {
	return sort.Search(len(s), func(i int) bool {
		return strings.Compare(s[i].String(), u.String()) <= 0
	})
}

// Has check if the URL is in this urlSet or not.
func (s urlSet) Has(u *url.URL) bool {
	i := s.search(u)
	if len(s) == i {
		return false
	}

	return s[i].String() == u.String()
}

// Add adds a URL to urlSet.
// If the URL is already added, it will be ignored.
func (s *urlSet) Add(u *url.URL) {
	i := s.search(u)
	if len(*s) == i {
		*s = append(*s, u)
	}

	if (*s)[i].String() != u.String() {
		*s = append(append((*s)[:i], u), (*s)[i:]...)
	}
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

	_, err = s.loadProbers(ctx)
	if errors.Is(err, ErrInvalidSourceURL) {
		return SourceProbe{}, err
	} else if err != nil {
		return SourceProbe{}, fmt.Errorf("%w: %s", ErrInvalidSource, err)
	}

	return s, nil
}

func (p SourceProbe) Target() *url.URL {
	return p.target
}

func openHTTPSource(ctx context.Context, u *url.URL) (io.ReadCloser, error) {
	ucopy := *u
	ucopy.Scheme = ucopy.Scheme[len("source+"):]
	resp, err := httpClient.Do((&http.Request{
		Method: "GET",
		URL:    &ucopy,
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
}

func openExecSource(ctx context.Context, u *url.URL) (io.ReadCloser, error) {
	var args []string
	if u.Fragment != "" {
		args = []string{u.Fragment}
	}

	cmd := exec.CommandContext(ctx, filepath.FromSlash(u.Opaque), args...)
	cmd.Env = getExecuteEnvByURL(u)

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
}

func openFileSource(ctx context.Context, u *url.URL) (io.ReadCloser, error) {
	f, err := os.Open(u.Opaque)
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

func openSource(ctx context.Context, u *url.URL) (io.ReadCloser, error) {
	switch u.Scheme {
	case "source+http", "source+https":
		return openHTTPSource(ctx, u)
	case "source+exec":
		return openExecSource(ctx, u)
	default:
		return openFileSource(ctx, u)
	}
}

func loadSource(ctx context.Context, target *url.URL, ignores *urlSet, fn func(u *url.URL) error) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	f, err := openSource(ctx, target)
	if err != nil {
		return err
	}
	defer f.Close()

	ignores.Add(target)

	invalids := &ayderr.ListBuilder{What: ErrInvalidSourceURL}

	scanner := &sourceScanner{Scanner: bufio.NewScanner(f)}
	for scanner.Scan() {
		u, err := scanner.URL()
		if err != nil {
			invalids.Pushf("%s", scanner.Text)
			continue
		}

		if u.Scheme != "source" {
			if !ignores.Has(u) {
				if err := fn(u); err != nil {
					invalids.Pushf("%s", u)
				}
				ignores.Add(u)
			}
			continue
		}

		if !ignores.Has(u) {
			err := loadSource(ctx, u, ignores, fn)
			es := ayderr.List{}
			if errors.As(err, &es) {
				invalids.Push(es.Children...)
			} else if err != nil {
				invalids.Pushf("%s", u)
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

func (p SourceProbe) loadProbers(ctx context.Context) ([]Prober, error) {
	var result []Prober

	err := loadSource(ctx, p.target, &urlSet{}, func(u *url.URL) error {
		p, err := NewProberFromURL(u)
		if err == nil {
			result = append(result, p)
		}
		return err
	})

	return result, err
}

func (p SourceProbe) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stime := time.Now()

	probes, err := p.loadProbers(ctx)
	if err != nil {
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
