package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type Certificate struct {
	PrivateKey  *rsa.PrivateKey
	CertFile    string
	KeyFile     string
	Certificate *x509.Certificate
}

func (c Certificate) Client() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // TODO: test certificate
			},
		},
	}
}

func generateSerialNumber(t *testing.T) *big.Int {
	i, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("failed to generate serial number: %s", err)
	}
	return i
}

func generatePrivateKey(t *testing.T) *rsa.PrivateKey {
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("failed to generate private key: %s", err)
	}
	return priv
}

func NewCertificate(t *testing.T) Certificate {
	baseDir := t.TempDir()

	c := Certificate{
		PrivateKey: generatePrivateKey(t),
		CertFile:   filepath.Join(baseDir, "cert.pem"),
		KeyFile:    filepath.Join(baseDir, "key.pem"),
		Certificate: &x509.Certificate{
			DNSNames:     []string{"localhost"},
			SerialNumber: generateSerialNumber(t),
			Subject: pkix.Name{
				Organization: []string{"Ayd"},
			},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().Add(3 * time.Hour),
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		},
	}

	der, err := x509.CreateCertificate(rand.Reader, c.Certificate, c.Certificate, &c.PrivateKey.PublicKey, c.PrivateKey)
	if err != nil {
		t.Fatalf("failed to generate certificate: %s", err)
	}

	certOut, err := os.Create(c.CertFile)
	defer certOut.Close()
	if err != nil {
		t.Fatalf("failed to create cert file: %s", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("failed to write cert file: %s", err)
	}

	keyOut, err := os.Create(c.KeyFile)
	defer keyOut.Close()
	if err != nil {
		t.Fatalf("failed to key cert file: %s", err)
	}
	priv, err := x509.MarshalPKCS8PrivateKey(c.PrivateKey)
	if err != nil {
		t.Fatalf("failed to marshal private key: %s", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: priv}); err != nil {
		t.Fatalf("failed to write cert file: %s", err)
	}

	return c
}
