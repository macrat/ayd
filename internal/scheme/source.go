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
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/macrat/ayd/internal/ayderr"
	"github.com/macrat/ayd/internal/scheme/textdecode"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/pkg/sftp"
)

var (
	ErrInvalidSource    = errors.New("invalid source")
	ErrInvalidSourceURL = errors.New("invalid source URL")
	ErrMissingFile      = errors.New("missing file")
)

func normalizeSourceURL(u *api.URL) (*api.URL, error) {
	switch u.Scheme {
	case "source+http", "source+https":
		if u.ToURL().Hostname() == "" {
			return nil, ErrMissingHost
		}
		if u.Path == "" {
			u.Path = "/"
		}
		return u, nil
	case "source+ftp", "source+ftps":
		if u.ToURL().Hostname() == "" {
			return nil, ErrMissingHost
		}
		if u.Path == "" || u.Path == "/" {
			return nil, ErrMissingFile
		}
		return &api.URL{
			Scheme:   u.Scheme,
			Host:     u.Host,
			Path:     path.Clean(u.Path),
			Fragment: u.Fragment,
		}, nil
	case "source+exec":
		p := u.Opaque
		if u.Opaque == "" {
			p = u.Path

			if p == "" || p == "/" {
				return nil, ErrMissingCommand
			}
		}
		return &api.URL{
			Scheme:   "source+exec",
			Opaque:   filepath.ToSlash(p),
			RawQuery: u.RawQuery,
			Fragment: u.Fragment,
		}, nil
	case "source+ssh":
		q := u.ToURL().Query()
		if f := u.ToURL().Query().Get("fingerprint"); f != "" {
			q.Set("fingerprint", strings.ReplaceAll(f, " ", "+"))
		}
		u := &api.URL{
			Scheme:   "source+ssh",
			User:     u.User,
			Host:     u.Host,
			Path:     u.Path,
			RawQuery: q.Encode(),
			Fragment: u.Fragment,
		}
		if _, err := newSSHConfig(u); err != nil {
			return nil, err
		}
		if u.Path == "" || u.Path == "/" {
			return nil, ErrMissingCommand
		}
		return u, nil
	case "source+sftp":
		q := u.ToURL().Query()
		if f := u.ToURL().Query().Get("fingerprint"); f != "" {
			q.Set("fingerprint", strings.ReplaceAll(f, " ", "+"))
		}
		u := &api.URL{
			Scheme:   "source+sftp",
			User:     u.User,
			Host:     u.Host,
			Path:     u.Path,
			RawQuery: q.Encode(),
			Fragment: u.Fragment,
		}
		if _, err := newSSHConfig(u); err != nil {
			return nil, err
		}
		if u.Path == "" || u.Path == "/" {
			return nil, ErrMissingCommand
		}
		return u, nil
	case "source":
		p := u.Opaque
		if u.Opaque == "" {
			p = u.Path

			if p == "" {
				return nil, ErrMissingFile
			}
		}
		return &api.URL{
			Scheme:   "source",
			Opaque:   filepath.ToSlash(filepath.Clean(p)),
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

func (s *sourceScanner) URL() (*api.URL, error) {
	u, err := api.ParseURL(s.Text)
	if err != nil {
		return nil, err
	}

	if s, _, _ := SplitScheme(u.Scheme); s == "source" {
		return normalizeSourceURL(u)
	}

	return u, nil
}

// SourceScheme implements how to load target URLs from file, HTTP, or external command.
type SourceScheme struct {
	target  *api.URL
	tracker *TargetTracker
}

func newSourceScheme(u *api.URL) (SourceScheme, error) {
	var err error
	u, err = normalizeSourceURL(u)
	if err != nil {
		return SourceScheme{}, err
	}

	s := SourceScheme{
		target:  u,
		tracker: &TargetTracker{},
	}

	return s, nil
}

func sourceLoadTest(_ interface{}, err error) error {
	if errors.Is(err, ErrInvalidSourceURL) {
		return err
	} else if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidSource, err)
	}

	return nil
}

// NewSourceProbe makes a new SourceScheme instance.
// It checks each URLs in source as a Prober.
func NewSourceProbe(u *api.URL) (SourceScheme, error) {
	s, err := newSourceScheme(u)
	if err != nil {
		return SourceScheme{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return s, sourceLoadTest(s.loadProbers(ctx))
}

// NewSourceAlert makes a new SourceScheme instance.
// It checks each URLs in source as an Alerter.
func NewSourceAlert(u *api.URL) (SourceScheme, error) {
	s, err := newSourceScheme(u)
	if err != nil {
		return SourceScheme{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return s, sourceLoadTest(s.loadAlerters(ctx))
}

func (p SourceScheme) Target() *api.URL {
	return p.target
}

func openHTTPSource(ctx context.Context, u *api.URL) (io.ReadCloser, error) {
	var ucopy url.URL = *u.ToURL()
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

func openFTPSource(ctx context.Context, u *api.URL) (io.ReadCloser, error) {
	ucopy := *u
	ucopy.Scheme = ucopy.Scheme[len("source+"):]
	conn, _, msg := ftpConnectAndLogin(ctx, &ucopy)
	if msg != "" {
		return nil, errors.New(msg)
	}
	defer conn.Quit()

	return conn.Retr(u.Path)
}

func openExecSource(ctx context.Context, u *api.URL) (io.ReadCloser, error) {
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
		msg, err := textdecode.Bytes(stderr.Bytes())
		if err != nil {
			msg = stderr.String()
		}
		return nil, fmt.Errorf("%w: failed to execute: %s", ErrInvalidURL, msg)
	}

	output, err := textdecode.Bytes(stdout.Bytes())
	if err != nil {
		return nil, err
	}
	return io.NopCloser(strings.NewReader(output)), nil
}

func openSSHSource(ctx context.Context, u *api.URL) (io.ReadCloser, error) {
	conf, err := newSSHConfig(u)
	if err != nil {
		return nil, err
	}

	conn, err := dialSSH(ctx, conf)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	env := make(map[string]string)
	for k, v := range u.ToURL().Query() {
		env[k] = v[len(v)-1]
	}

	output, _, _, err := conn.Exec(ctx, u.Path, u.Fragment, env)
	return io.NopCloser(strings.NewReader(output)), nil
}

func openSFTPSource(ctx context.Context, u *api.URL) (io.ReadCloser, error) {
	conf, err := newSSHConfig(u)
	if err != nil {
		return nil, err
	}

	ssh, err := dialSSH(ctx, conf)
	if err != nil {
		return nil, err
	}

	conn, err := sftp.NewClient(ssh.Client)
	if err != nil {
		return nil, err
	}

	return conn.OpenFile(u.Path, os.O_RDONLY)
}

func openFileSource(ctx context.Context, u *api.URL) (io.ReadCloser, error) {
	raw, err := os.ReadFile(u.Opaque)
	if err != nil {
		return nil, err
	}

	s, err := textdecode.Bytes(raw)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(strings.NewReader(s)), nil
}

func openSource(ctx context.Context, u *api.URL) (io.ReadCloser, error) {
	switch u.Scheme {
	case "source+http", "source+https":
		return openHTTPSource(ctx, u)
	case "source+ftp", "source+ftps":
		return openFTPSource(ctx, u)
	case "source+exec":
		return openExecSource(ctx, u)
	case "source+ssh":
		return openSSHSource(ctx, u)
	case "source+sftp":
		return openSFTPSource(ctx, u)
	default:
		return openFileSource(ctx, u)
	}
}

func loadSource(ctx context.Context, target *api.URL, ignores *urlSet, fn func(u *api.URL) (normalized *api.URL, err error)) error {
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

		if ignores.Has(u) {
			continue
		}

		if s, _, _ := SplitScheme(u.Scheme); s != "source" {
			u2, err := fn(u)
			if err != nil {
				invalids.Pushf("%s", u)
			} else {
				ignores.Add(u2)
			}
		} else {
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

func (p SourceScheme) loadProbers(ctx context.Context) ([]Prober, error) {
	var result []Prober

	err := loadSource(ctx, p.target, &urlSet{}, func(u *api.URL) (*api.URL, error) {
		p, err := NewProberFromURL(u)
		if err == nil {
			result = append(result, p)
		}
		return p.Target(), err
	})

	return result, err
}

func (p SourceScheme) loadAlerters(ctx context.Context) ([]Alerter, error) {
	var result []Alerter

	err := loadSource(ctx, p.target, &urlSet{}, func(u *api.URL) (*api.URL, error) {
		p, err := NewAlerterFromURL(u)
		if err == nil {
			result = append(result, p)
		}
		return p.Target(), err
	})

	return result, err
}

func (p SourceScheme) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stime := time.Now()

	probes, err := p.loadProbers(ctx)
	d := time.Since(stime)

	if err != nil {
		r.Report(p.target, timeoutOr(ctx, api.Record{
			Time:    stime,
			Latency: d,
			Status:  api.StatusFailure,
			Target:  p.target,
			Message: err.Error(),
		}))
		return
	}

	r.Report(p.target, api.Record{
		Time:    stime,
		Status:  api.StatusHealthy,
		Latency: d,
		Target:  p.target,
		Message: fmt.Sprintf("loaded %d targets", len(probes)),
		Extra: map[string]interface{}{
			"target_count": len(probes),
		},
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

func (p SourceScheme) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	r = AlertReporter{p.target, r}

	stime := time.Now()

	alerters, err := p.loadAlerters(ctx)
	d := time.Since(stime)

	if err != nil {
		r.Report(p.target, timeoutOr(ctx, api.Record{
			Time:    stime,
			Latency: d,
			Status:  api.StatusFailure,
			Target:  p.target,
			Message: err.Error(),
		}))
		return
	}

	r.Report(p.target, api.Record{
		Time:    stime,
		Latency: d,
		Status:  api.StatusHealthy,
		Target:  p.target,
		Message: fmt.Sprintf("loaded %d targets", len(alerters)),
		Extra: map[string]interface{}{
			"target_count": len(alerters),
		},
	})

	wg := &sync.WaitGroup{}
	for _, a := range alerters {
		wg.Add(1)

		go func(a Alerter) {
			a.Alert(ctx, r, lastRecord)
			wg.Done()
		}(a)
	}
	wg.Wait()
}
