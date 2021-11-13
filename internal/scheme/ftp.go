package scheme

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path"
	"time"

	"github.com/jlaffaye/ftp"
	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrMissingUsername = errors.New("username is required if set password")
	ErrMissingPassword = errors.New("password is required if set username")
)

// FTPProbe is a implementation for the FTP.
type FTPProbe struct {
	target *url.URL
}

func NewFTPProbe(u *url.URL) (FTPProbe, error) {
	p := FTPProbe{
		target: &url.URL{
			Scheme:   u.Scheme,
			User:     u.User,
			Host:     u.Host,
			Path:     path.Clean(u.Path),
			Fragment: u.Fragment,
		},
	}

	if u.Host == "" {
		return FTPProbe{}, ErrMissingHost
	}

	if u.User != nil {
		if u.User.Username() == "" {
			return FTPProbe{}, ErrMissingUsername
		}
		if _, ok := u.User.Password(); !ok {
			return FTPProbe{}, ErrMissingPassword
		}
	}

	if u.Path == "" {
		p.target.Path = "/"
	}

	return p, nil
}

func (p FTPProbe) Target() *url.URL {
	return p.target
}

func (p FTPProbe) host() string {
	if p.target.Port() != "" {
		return p.target.Host
	}
	return p.target.Host + ":21"
}

func (p FTPProbe) options(ctx context.Context) []ftp.DialOption {
	opts := []ftp.DialOption{
		ftp.DialWithContext(ctx),
	}
	if p.target.Scheme == "ftps" {
		opts = append(opts, ftp.DialWithExplicitTLS(&tls.Config{}))
	}
	return opts
}

func (p FTPProbe) userInfo() (user, pass string) {
	if p.target.User == nil {
		return "anonymous", "anonymous"
	}

	user = p.target.User.Username()
	pass, _ = p.target.User.Password()

	return user, pass
}

func (p FTPProbe) dial(ctx context.Context) (conn *ftp.ServerConn, status api.Status, message string) {
	conn, err := ftp.Dial(p.host(), p.options(ctx)...)
	if err == nil {
		return conn, api.StatusHealthy, ""
	}

	status = api.StatusFailure
	message = err.Error()

	dnsErr := &net.DNSError{}
	opErr := &net.OpError{}

	if errors.As(err, &dnsErr) {
		status = api.StatusUnknown
		message = dnsErrorToMessage(dnsErr)
	} else if errors.As(err, &opErr) && opErr.Op == "dial" {
		message = fmt.Sprintf("%s: connection refused", opErr.Addr)
	}

	return
}

func (p FTPProbe) login(conn *ftp.ServerConn) (status api.Status, message string) {
	if err := conn.Login(p.userInfo()); err != nil {
		return api.StatusFailure, err.Error()
	}
	return api.StatusHealthy, ""
}

func (p FTPProbe) list(conn *ftp.ServerConn) (files []*ftp.Entry, status api.Status, message string) {
	ls, err := conn.List(p.target.Path)
	if err != nil {
		return nil, api.StatusFailure, err.Error()
	}
	if len(ls) == 0 {
		return nil, api.StatusFailure, "no such file or directory"
	}
	return ls, api.StatusHealthy, ""
}

func (p FTPProbe) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	stime := time.Now()
	report := func(status api.Status, message string) {
		r.Report(p.target, timeoutOr(ctx, api.Record{
			CheckedAt: stime,
			Status:    status,
			Latency:   time.Since(stime),
			Target:    p.target,
			Message:   message,
		}))
	}

	conn, status, message := p.dial(ctx)
	if status != api.StatusHealthy {
		report(status, message)
		return
	}
	defer conn.Quit()

	if status, message = p.login(conn); status != api.StatusHealthy {
		report(status, message)
		return
	}

	ls, status, message := p.list(conn)
	if status != api.StatusHealthy {
		report(status, message)
		return
	}

	n := 0
	for _, f := range ls {
		if f.Name != "." && f.Name != ".." {
			n++
		}
	}

	if n == 1 && ls[0].Name == path.Base(p.target.Path) {
		report(api.StatusHealthy, fmt.Sprintf("type=file size=%d", ls[0].Size))
	} else {
		report(api.StatusHealthy, fmt.Sprintf("type=directory files=%d", n))
	}
}
