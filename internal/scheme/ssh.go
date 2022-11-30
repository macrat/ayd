package scheme

import (
	"net"
	"os"
	"net/url"
	"strings"
	"errors"
	"time"
	"context"

	"golang.org/x/crypto/ssh"
	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrInvalidFingerprint     = errors.New("invalid fingerprint format")
	ErrFingerprintUnmatched   = errors.New("fingerprint unmatched")
)

type sshConfig struct {
	Host string
	User string
	Auth []ssh.AuthMethod
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
		if err != nil {
			return c, err
		}

		var signer ssh.Signer
		if p, ok := u.User.Password(); ok {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(pem, []byte(p))
		} else {
			signer, err = ssh.ParsePrivateKey(pem)
		}
		if err != nil {
			return c, err
		}

		c.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	} else if password, ok := u.User.Password(); ok {
		c.Auth = []ssh.AuthMethod{
			ssh.Password(password),
		}
	}

	if fingerprint := query.Get("fingerprint"); fingerprint != "" {
		switch {
		case strings.HasPrefix(strings.ToUpper(fingerprint), "SHA256:"):
			c.CheckKey = func(key ssh.PublicKey) bool {
				return ssh.FingerprintSHA256(key) == fingerprint
			}
		case strings.HasPrefix(strings.ToUpper(fingerprint), "MD5:"):
			c.CheckKey = func(key ssh.PublicKey) bool {
				return ssh.FingerprintLegacyMD5(key) == fingerprint
			}
		default:
			return c, ErrInvalidFingerprint
		}
	} else {
		c.CheckKey = func(key ssh.PublicKey) bool {
			return true
		}
	}

	return c, nil
}

type sshConnection struct {
	Client *ssh.Client
	Banner string
	Fingerprint string
}

func (conn sshConnection) Close() error {
	return conn.Client.Close()
}

func dialSSH(ctx context.Context, c sshConfig) (conn sshConnection, err error) {
	timeout := 5 * time.Minute
	if t, ok := ctx.Deadline(); ok {
		x := time.Until(t)
		if x < timeout {
			timeout = x
		}
	}

	conn.Client, err = ssh.Dial("tcp", c.Host, &ssh.ClientConfig{
		User: c.User,
		Auth: c.Auth,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			conn.Fingerprint = ssh.FingerprintSHA256(key)
			if !c.CheckKey(key) {
				return ErrFingerprintUnmatched
			}
			return nil
		},
		BannerCallback: func(msg string) error {
			conn.Banner = msg
			return nil
		},
		Timeout: timeout,
	})
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
	if f := u.ToURL().Query().Get("fingerprint"); f != "" {
		q.Set("fingerprint", f)
	}

	conf, err := newSSHConfig(u)
	if err != nil {
		return SSHProbe{}, err
	}

	u = &api.URL{
		Scheme: "ssh",
		User: url.User(u.User.Username()),
		Host: strings.ToLower(u.Host),
		RawQuery: q.Encode(),
		Fragment: u.Fragment,
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
	rec := api.Record{
		Time: time.Now(),
		Target: s.target,
		Status: api.StatusHealthy,
	}

	conn, err := dialSSH(ctx, s.conf)
	rec.Latency = time.Since(rec.Time)

	if err != nil {
		rec.Status = api.StatusFailure
		rec.Message = err.Error()
		r.Report(s.target, rec)
		return
	}

	rec.Message = conn.Banner
	rec.Extra = map[string]interface{}{
		"fingerprint": conn.Fingerprint,
		"source_addr": conn.Client.LocalAddr().String(),
		"target_addr": conn.Client.RemoteAddr().String(),
	}
	r.Report(s.target, rec)

	conn.Close()
}
