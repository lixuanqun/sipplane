package dataplane_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sipplane/sipplane/internal/config"
	"github.com/sipplane/sipplane/internal/dataplane"
)

func TestTLSListenRequiresCert(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	httpAddr := ln.Addr().String()
	_ = ln.Close()

	sipPort := freeTCPPort(t)
	cfg := config.Config{
		Listen:         "127.0.0.1:" + itoa(sipPort),
		Transport:      "tls",
		AdvertisedHost: "127.0.0.1",
		AdvertisedPort: sipPort,
		HTTPListen:     httpAddr,
		Realm:          "sipplane",
		LogLevel:       "error",
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := dataplane.New(cfg, labSnapshot(), log)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = srv.Run(ctx)
	if err == nil {
		t.Fatal("expected error when tls cert missing")
	}
}

func TestTLSListenAndHandshake(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	if err := writeSelfSigned(certFile, keyFile); err != nil {
		t.Fatal(err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	httpAddr := ln.Addr().String()
	_ = ln.Close()

	sipPort := freeTCPPort(t)
	cfg := config.Config{
		Listen:         "127.0.0.1:" + itoa(sipPort),
		Transport:      "tls",
		AdvertisedHost: "127.0.0.1",
		AdvertisedPort: sipPort,
		HTTPListen:     httpAddr,
		Realm:          "sipplane",
		LogLevel:       "error",
		TLSCertFile:    certFile,
		TLSKeyFile:     keyFile,
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := dataplane.New(cfg, labSnapshot(), log)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()
	waitHTTP(t, "http://"+httpAddr+"/readyz")

	deadline := time.Now().Add(3 * time.Second)
	var dialErr error
	for time.Now().Before(deadline) {
		conn, err := tls.DialWithDialer(
			&net.Dialer{Timeout: 500 * time.Millisecond},
			"tcp",
			cfg.Listen,
			&tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12},
		)
		if err == nil {
			_ = conn.Close()
			cancel()
			select {
			case <-errCh:
			case <-time.After(2 * time.Second):
			}
			return
		}
		dialErr = err
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	t.Fatalf("tls handshake failed: %v", dialErr)
}

func writeSelfSigned(certPath, keyPath string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "sipplane-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		_ = certOut.Close()
		return err
	}
	_ = certOut.Close()

	keyOut, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		_ = keyOut.Close()
		return err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}); err != nil {
		_ = keyOut.Close()
		return err
	}
	return keyOut.Close()
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}
