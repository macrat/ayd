package scheme

import (
	"bytes"
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
		} else if errors.As(err, &opErr) && opErr.Op == "dial" && opErr.Addr != nil {
			message = fmt.Sprintf("%s: connection refused", opErr.Addr)
		}

		return
	}

	if err := conn.Login(ftpUserInfo(u)); err != nil {
		conn.Quit()
		return nil, api.StatusFailure, err.Error()
	}

	return conn, api.StatusHealthy, ""
}

// FTPScheme is a probe/alert implementation for the FTP.
type FTPScheme struct {
	target *api.URL
}

func NewFTPScheme(u *api.URL) (FTPScheme, error) {
	s := FTPScheme{
		target: &api.URL{
			Scheme:   u.Scheme,
			User:     u.User,
			Host:     u.Host,
			Path:     path.Clean(u.Path),
			Fragment: u.Fragment,
		},
	}

	if u.Host == "" {
		return FTPScheme{}, ErrMissingHost
	}

	if u.User != nil {
		if u.User.Username() == "" {
			return FTPScheme{}, ErrMissingUsername
		}
		if _, ok := u.User.Password(); !ok {
			return FTPScheme{}, ErrMissingPassword
		}
	}

	if u.Path == "" {
		s.target.Path = "/"
	}

	return s, nil
}

func (s FTPScheme) Target() *api.URL {
	return s.target
}

func (s FTPScheme) list(conn *ftp.ServerConn) (files []*ftp.Entry, status api.Status, message string) {
	ls, err := conn.List(s.target.Path)
	if err != nil {
		return nil, api.StatusFailure, err.Error()
	}
	if len(ls) == 0 {
		return nil, api.StatusFailure, "no such file or directory"
	}
	return ls, api.StatusHealthy, ""
}

// Probe checks if the target FTP server is available.
func (s FTPScheme) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	stime := time.Now()
	report := func(status api.Status, message string, extra map[string]interface{}) {
		r.Report(s.target, timeoutOr(ctx, api.Record{
			Time:    stime,
			Status:  status,
			Latency: time.Since(stime),
			Target:  s.target,
			Message: message,
			Extra:   extra,
		}))
	}

	conn, status, message := ftpConnectAndLogin(ctx, s.target)
	if status != api.StatusHealthy {
		report(status, message, nil)
		return
	}
	defer conn.Quit()

	ls, status, message := s.list(conn)
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

	if n == 1 && path.Base(ls[0].Name) == path.Base(s.target.Path) {
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

func (s FTPScheme) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	target := &api.URL{
		Scheme: "alert",
		Opaque: s.target.String(),
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	stime := time.Now()
	report := func(status api.Status, message string, extra map[string]interface{}) {
		r.Report(s.target, timeoutOr(ctx, api.Record{
			Time:    stime,
			Status:  status,
			Latency: time.Since(stime),
			Target:  target,
			Message: message,
			Extra:   extra,
		}))
	}

	conn, status, message := ftpConnectAndLogin(ctx, s.target)
	if status != api.StatusHealthy {
		report(status, message, nil)
		return
	}
	defer conn.Quit()

	line := lastRecord.String() + "\n"

	err := conn.Append(s.target.Path, bytes.NewBufferString(line))
	if err != nil {
		report(api.StatusFailure, fmt.Sprintf("failed to upload record: %s", err), nil)
	} else {
		report(api.StatusHealthy, fmt.Sprintf("uploaded %d bytes to the server", len(line)), nil)
	}
}
