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

// ftpOptions makes ftp.DialOptions for the ftp library.
//
func ftpOptions(ctx context.Context, u *url.URL) []ftp.DialOption {
	opts := []ftp.DialOption{
		ftp.DialWithDialFunc(func(network, address string) (net.Conn, error) {
			conn, err := (&net.Dialer{}).DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			go func() {
				select {
				case <-ctx.Done():
					conn.SetDeadline(time.Now())
				}
			}()
			return conn, nil
		}),
	}
	if u.Scheme == "ftps" {
		opts = append(opts, ftp.DialWithExplicitTLS(&tls.Config{}))
	}
	return opts
}

func ftpUserInfo(u *url.URL) (user, pass string) {
	if u.User == nil {
		return "anonymous", "anonymous"
	}

	user = u.User.Username()
	pass, _ = u.User.Password()

	return user, pass
}

// ftpConnectAndLogin makes FTP connection by URL.
//
// If the context timed out, it will terminate TCP connection without QUIT command to the server.
// This is not very graceful, but it can stop connection surely.
func ftpConnectAndLogin(ctx context.Context, u *url.URL) (conn *ftp.ServerConn, status api.Status, message string) {
	host := u.Host
	if u.Port() == "" {
		host += ":21"
	}

	conn, err := ftp.Dial(host, ftpOptions(ctx, u)...)
	if err != nil {
		status = api.StatusFailure
		message = err.Error()

		dnsErr := &net.DNSError{}
		opErr := &net.OpError{}

		if errors.As(err, &dnsErr) {
			status = api.StatusUnknown
			message = dnsErrorToMessage(dnsErr)
		} else if errors.As(err, &opErr) && opErr.Op == "dial" {
			if opErr.Addr == nil {
				message = err.Error()
			} else {
				message = fmt.Sprintf("%s: connection refused", opErr.Addr)
			}
		}

		return
	}

	if err := conn.Login(ftpUserInfo(u)); err != nil {
		conn.Quit()
		return nil, api.StatusFailure, err.Error()
	}

	return conn, api.StatusHealthy, ""
}

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

// Probe checks if the target FTP server is available.
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

	conn, status, message := ftpConnectAndLogin(ctx, p.target)
	if status != api.StatusHealthy {
		report(status, message)
		return
	}
	defer conn.Quit()

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
