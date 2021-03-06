package scheme

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
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
func ftpOptions(ctx context.Context, u *api.URL) []ftp.DialOption {
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

func ftpUserInfo(u *api.URL) (user, pass string) {
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
func ftpConnectAndLogin(ctx context.Context, u *api.URL) (conn *ftp.ServerConn, status api.Status, message string) {
	host := u.Host
	if u.ToURL().Port() == "" {
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
	target *api.URL
}

func NewFTPProbe(u *api.URL) (FTPProbe, error) {
	p := FTPProbe{
		target: &api.URL{
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

func (p FTPProbe) Target() *api.URL {
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
	report := func(status api.Status, message string, extra map[string]interface{}) {
		r.Report(p.target, timeoutOr(ctx, api.Record{
			Time:    stime,
			Status:  status,
			Latency: time.Since(stime),
			Target:  p.target,
			Message: message,
			Extra:   extra,
		}))
	}

	conn, status, message := ftpConnectAndLogin(ctx, p.target)
	if status != api.StatusHealthy {
		report(status, message, nil)
		return
	}
	defer conn.Quit()

	ls, status, message := p.list(conn)
	if status != api.StatusHealthy {
		report(status, message, nil)
		return
	}

	n := 0
	for _, f := range ls {
		if f.Name != "." && f.Name != ".." {
			n++
		}
	}

	if n == 1 && path.Base(ls[0].Name) == path.Base(p.target.Path) {
		report(api.StatusHealthy, "file exists", map[string]interface{}{
			"file_size": ls[0].Size,
			"mtime":     ls[0].Time.Format(time.RFC3339),
			"type":      "file",
		})
	} else {
		extra := map[string]interface{}{
			"file_count": n,
			"type":       "directory",
		}
		for _, f := range ls {
			if f.Name == "." {
				extra["mtime"] = f.Time.Format(time.RFC3339)
				break
			}
		}
		report(api.StatusHealthy, "directory exists", extra)
	}
}
