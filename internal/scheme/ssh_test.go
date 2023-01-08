package scheme_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
	"golang.org/x/crypto/ssh"
)

func GenerateSSHKey(t testing.TB) (*rsa.PrivateKey, ssh.PublicKey) {
	pri, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}

	pub, err := ssh.NewPublicKey(&pri.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	return pri, pub
}

func SaveSSHKey(t testing.TB, key *rsa.PrivateKey, name, passphrase string) (path string) {
	path = filepath.Join(t.TempDir(), name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	if passphrase != "" {
		block, err = x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(passphrase), x509.PEMCipherAES256)
		if err != nil {
			t.Fatal(err)
		}
	}

	pem.Encode(f, block)

	return path
}

type SSHServer struct {
	Addr           string
	BareKey        string
	EncryptedKey   string
	Listener       net.Listener
	Conf           *ssh.ServerConfig
	FingerprintSHA string
	FingerprintMD5 string
	Stop           context.CancelFunc
}

func (s SSHServer) Close() error {
	s.Stop()
	return s.Listener.Close()
}

func (s SSHServer) Serve(ctx context.Context) {
	for {
		tcpConn, err := s.Listener.Accept()
		if err != nil {
			break
		}

		_, chans, reqs, err := ssh.NewServerConn(tcpConn, s.Conf)
		if err != nil {
			continue
		}

		go ssh.DiscardRequests(reqs)

		go func() {
			for newChannel := range chans {
				if newChannel.ChannelType() != "session" {
					newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
					continue
				}
				channel, requests, err := newChannel.Accept()
				if err != nil {
					return
				}

				go func(in <-chan *ssh.Request) {
					for req := range in {
						switch req.Type {
						case "env":
							req.Reply(true, nil)

							var env struct {
								Key   string
								Value string
							}
							if err := ssh.Unmarshal(req.Payload, &env); err == nil {
								fmt.Fprintf(channel, "env %s=%s\n", env.Key, env.Value)
							}

						case "exec":
							req.Reply(true, nil)

							cmd := string(req.Payload[4:])
							fmt.Fprintf(channel, "exec %s", cmd)

							var status struct {
								Status uint32
							}
							if cmd == `"/error"` {
								status.Status = 1
							} else if cmd == `"/not-found"` {
								status.Status = 127
							}

							if cmd != `"/crash"` {
								channel.SendRequest("exit-status", false, ssh.Marshal(&status))
							}
							channel.Close()

						default:
							req.Reply(false, nil)
						}
					}
				}(requests)
			}
		}()
	}
}

func StartSSHServer(t testing.TB) SSHServer {
	ctx, stop := context.WithCancel(context.Background())

	pri, pub := GenerateSSHKey(t)
	pubfinger := ssh.FingerprintSHA256(pub)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listen: %s", err)
	}

	srvPri, srvPub := GenerateSSHKey(t)
	signer, err := ssh.NewSignerFromKey(srvPri)
	if err != nil {
		t.Fatalf("failed to generate signer: %s", err)
	}

	conf := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if conn.User() == "keyusr" && ssh.FingerprintSHA256(key) == pubfinger {
				return &ssh.Permissions{}, nil
			}
			return nil, errors.New("failed to auth")
		},
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if conn.User() == "pasusr" && string(password) == "foobar" {
				return &ssh.Permissions{}, nil
			}
			return nil, errors.New("failed to auth")
		},
	}
	conf.AddHostKey(signer)

	srv := SSHServer{
		Addr:           listener.Addr().String(),
		BareKey:        SaveSSHKey(t, pri, "bare_rsa", ""),
		EncryptedKey:   SaveSSHKey(t, pri, "enc_rsa", "helloworld"),
		Listener:       listener,
		Conf:           conf,
		FingerprintSHA: ssh.FingerprintSHA256(srvPub),
		FingerprintMD5: "MD5:" + ssh.FingerprintLegacyMD5(srvPub),
		Stop:           stop,
	}

	go srv.Serve(ctx)

	return srv
}

func TestSSHProbe_Probe(t *testing.T) {
	t.Parallel()

	server := StartSSHServer(t)

	extra := fmt.Sprintf("---\nfingerprint: %s\nsource_addr: [^ ]+\ntarget_addr: %s", regexp.QuoteMeta(server.FingerprintSHA), server.Addr)
	success := "succeed to connect\n" + extra
	failedToAuth := func(method string) string {
		return fmt.Sprintf(`ssh: handshake failed: ssh: unable to authenticate, attempted methods \[none %s\], no supported methods remain`+"\n%s", method, extra)
	}

	dummyKey, _ := GenerateSSHKey(t)
	dummyPath := SaveSSHKey(t, dummyKey, "dummy_rsa", "")

	AssertProbe(t, []ProbeTest{
		{"ssh://" + server.Addr, api.StatusUnknown, success, "username is required"},
		{"ssh://pasusr:foobar@" + server.Addr, api.StatusHealthy, success, ""},
		{"ssh://pasusr:foobar@" + server.Addr + "?fingerprint=" + url.QueryEscape(server.FingerprintSHA), api.StatusHealthy, success, ""},
		{"ssh://pasusr:foobar@" + server.Addr + "?fingerprint=" + url.QueryEscape(server.FingerprintMD5), api.StatusHealthy, success, ""},
		{"ssh://pasusr:foobar@" + server.Addr + "?fingerprint=SHA256%3AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", api.StatusFailure, "ssh: handshake failed: fingerprint unmatched\n" + extra, ""},
		{"ssh://pasusr:foobar@" + server.Addr + "?fingerprint=MD5%3AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", api.StatusFailure, "ssh: handshake failed: fingerprint unmatched\n" + extra, ""},
		{"ssh://pasusr:invalid@" + server.Addr, api.StatusFailure, failedToAuth("password"), ""},
		{"ssh://keyusr@" + server.Addr + "?identityfile=" + url.QueryEscape(server.BareKey), api.StatusHealthy, success, ""},
		{"ssh://keyusr:helloworld@" + server.Addr + "?identityfile=" + url.QueryEscape(server.EncryptedKey), api.StatusHealthy, success, ""},
		{"ssh://keyusr@" + server.Addr + "?fingerprint=" + url.QueryEscape(server.FingerprintSHA) + "&identityfile=" + url.QueryEscape(server.BareKey), api.StatusHealthy, success, ""},
		{"ssh://keyusr@" + server.Addr + "?fingerprint=" + url.QueryEscape(server.FingerprintMD5) + "&identityfile=" + url.QueryEscape(server.BareKey), api.StatusHealthy, success, ""},
		{"ssh://keyusr@" + server.Addr + "?identityfile=" + url.QueryEscape(dummyPath), api.StatusFailure, failedToAuth("publickey"), ""},
		{"ssh://keyusr@" + server.Addr + "?identityfile=testdata%2Ffile.txt", api.StatusUnknown, "", "invalid identity file: testdata/file.txt"},
		{"ssh://keyusr@" + server.Addr + "?identityfile=testdata%2Fno-such-file", api.StatusUnknown, "", "no such identity file: testdata/no-such-file"},
		{"ssh://keyusr:incorrect@" + server.Addr + "?identityfile=" + url.QueryEscape(server.EncryptedKey), api.StatusUnknown, "", "identity file: x509: decryption password incorrect"},
		{"ssh://keyusr@" + server.Addr + "?identityfile=" + url.QueryEscape(server.EncryptedKey), api.StatusUnknown, "", "identity file: ssh: this private key is passphrase protected"},
		{"ssh://someone@" + server.Addr, api.StatusUnknown, success, "password or identityfile is required"},
		{"ssh://foo:bar@" + server.Addr + "?fingerprint=abc", api.StatusUnknown, success, "unsupported fingerprint format"},
	}, 10)

	AssertTimeout(t, "ssh://pasusr:foobar@"+server.Addr)
}
