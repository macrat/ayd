package scheme

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"golang.org/x/crypto/ssh"
)

type sshConfig struct {
	Host     string
	User     string
	Auth     []ssh.AuthMethod
	CheckKey func(ssh.PublicKey) (ok bool)
}

func newSSHConfig(u *api.URL) (sshConfig, error) {
	c := sshConfig{
		Host: u.Host,
	}
	if u.ToURL().Port() == "" {
		c.Host += ":22"
	}

	if u.User == nil {
		return c, errors.New("username is required")
	}
	c.User = u.User.Username()

	query := u.ToURL().Query()

	if identityFile := query.Get("identityfile"); identityFile != "" {
		pem, err := os.ReadFile(identityFile)
		if errors.Is(err, os.ErrNotExist) {
			return c, fmt.Errorf("no such identity file: %s", identityFile)
		} else if err != nil {
			return c, err
		}

		var signer ssh.Signer
		if p, ok := u.User.Password(); ok {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(pem, []byte(p))
		} else {
			signer, err = ssh.ParsePrivateKey(pem)
		}
		if err != nil {
			if err.Error() == "ssh: no key found" {
				return c, fmt.Errorf("invalid identity file: %s", identityFile)
			}
			return c, fmt.Errorf("identity file: %w", err)
		}

		c.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	} else if password, ok := u.User.Password(); ok {
		c.Auth = []ssh.AuthMethod{
			ssh.Password(password),
		}
	} else {
		return c, errors.New("password or identityfile is required")
	}

	if fingerprint := query.Get("fingerprint"); fingerprint != "" {
		switch {
		case strings.HasPrefix(fingerprint, "SHA256:"):
			c.CheckKey = func(key ssh.PublicKey) bool {
				return ssh.FingerprintSHA256(key) == fingerprint
			}
		case strings.HasPrefix(fingerprint, "MD5:"):
			fingerprint := strings.ToLower(fingerprint)[len("MD5:"):]
			c.CheckKey = func(key ssh.PublicKey) bool {
				return ssh.FingerprintLegacyMD5(key) == fingerprint
			}
		default:
			return c, errors.New("unsupported fingerprint format")
		}
	} else {
		c.CheckKey = func(key ssh.PublicKey) bool {
			return true
		}
	}

	return c, nil
}

type sshConnection struct {
	Client      *ssh.Client
	Fingerprint string
	SourceAddr  string
	TargetAddr  string
}

func (conn sshConnection) Close() error {
	if conn.Client != nil {
		return conn.Client.Close()
	}
	return nil
}

func (conn sshConnection) MakeExtra() map[string]any {
	extra := make(map[string]any)
	if conn.Fingerprint != "" {
		extra["fingerprint"] = conn.Fingerprint
	}
	if conn.SourceAddr != "" {
		extra["source_addr"] = conn.SourceAddr
	}
	if conn.TargetAddr != "" {
		extra["target_addr"] = conn.TargetAddr
	}
	return extra
}

func dialSSH(ctx context.Context, c sshConfig) (conn sshConnection, err error) {
	var dialer net.Dialer
	rawConn, err := dialer.DialContext(ctx, "tcp", c.Host)
	if err != nil {
		return conn, err
	}

	timeout := 10 * time.Minute
	if t, ok := ctx.Deadline(); ok {
		x := time.Until(t)
		if x < timeout {
			timeout = x
		}
	}

	conn.SourceAddr = rawConn.LocalAddr().String()
	conn.TargetAddr = rawConn.RemoteAddr().String()

	sshConn, chans, reqs, err := ssh.NewClientConn(rawConn, c.Host, &ssh.ClientConfig{
		User: c.User,
		Auth: c.Auth,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			conn.Fingerprint = ssh.FingerprintSHA256(key)
			if !c.CheckKey(key) {
				return errors.New("fingerprint unmatched")
			}
			return nil
		},
		Timeout: timeout,
	})
	if err != nil {
		rawConn.Close()
		return conn, err
	}

	conn.Client = ssh.NewClient(sshConn, chans, reqs)

	return
}

// SSHProbe is a Prober implementation for SSH protocol.
type SSHProbe struct {
	target *api.URL
	conf   sshConfig
}

func NewSSHProbe(u *api.URL) (SSHProbe, error) {
	_, separator, _ := SplitScheme(u.Scheme)
	if separator != 0 {
		return SSHProbe{}, ErrUnsupportedScheme
	}

	q := url.Values{}
	if f := u.ToURL().Query().Get("identityfile"); f != "" {
		q.Set("identityfile", f)
	}
	if f := u.ToURL().Query().Get("fingerprint"); f != "" {
		q.Set("fingerprint", strings.ReplaceAll(f, " ", "+"))
	}

	u = &api.URL{
		Scheme:   "ssh",
		User:     u.User,
		Host:     strings.ToLower(u.Host),
		RawQuery: q.Encode(),
		Fragment: u.Fragment,
	}

	conf, err := newSSHConfig(u)
	if err != nil {
		return SSHProbe{}, err
	}

	return SSHProbe{
		target: u,
		conf:   conf,
	}, nil
}

func (s SSHProbe) Target() *api.URL {
	return s.target
}

func (s SSHProbe) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	rec := api.Record{
		Time:   time.Now(),
		Target: s.target,
		Status: api.StatusHealthy,
	}

	conn, err := dialSSH(ctx, s.conf)
	rec.Latency = time.Since(rec.Time)
	conn.Close()

	var dnsErr *net.DNSError
	var opErr *net.OpError
	if errors.As(err, &dnsErr) {
		rec.Status = api.StatusUnknown
		rec.Message = dnsErrorToMessage(dnsErr)
	} else if errors.As(err, &opErr) && opErr.Op == "dial" {
		rec.Status = api.StatusFailure
		if opErr.Addr == nil {
			rec.Message = err.Error()
		} else {
			rec.Message = fmt.Sprintf("%s: connection refused", opErr.Addr)
		}
	} else if err != nil {
		rec.Status = api.StatusFailure
		rec.Message = err.Error()
	} else {
		rec.Message = "succeed to connect"
	}

	rec.Extra = conn.MakeExtra()
	r.Report(s.target, timeoutOr(ctx, rec))
}
