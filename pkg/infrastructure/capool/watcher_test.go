package capool

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
	"log"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

const (
	testInterval = 20 * time.Millisecond
	testTimeout  = 5 * time.Second
)

type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pem  []byte
}

var serialNumber atomic.Int64

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

func newCA(t *testing.T, name string) *testCA {
	t.Helper()
	key := newKey(t)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(serialNumber.Add(1)),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create CA: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse CA: %v", err)
	}
	return &testCA{
		cert: cert,
		key:  key,
		pem:  pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
	}
}

// leaf returns a key pair signed by ca, for a client or a server on 127.0.0.1.
func (ca *testCA) leaf(t *testing.T, usage x509.ExtKeyUsage) tls.Certificate {
	t.Helper()
	key := newKey(t)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(serialNumber.Add(1)),
		Subject:      pkix.Name{CommonName: "leaf"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{usage},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: cert}
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func newWatcher(t *testing.T, path string) *Watcher {
	t.Helper()
	w, err := New(path, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return w.WithInterval(testInterval)
}

// startWatcher runs w until the end of the test.
func startWatcher(t *testing.T, w *Watcher) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Start(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Start: %v", err)
		}
	})
}

func clientState(cert tls.Certificate) tls.ConnectionState {
	return tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert.Leaf}}
}

func eventually(t *testing.T, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(testTimeout)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal(message)
		}
		time.Sleep(testInterval)
	}
}

func TestNewErrors(t *testing.T) {
	dir := t.TempDir()

	missing := filepath.Join(dir, "missing.crt")
	if _, err := New(missing, slog.New(slog.DiscardHandler)); !errors.Is(err, domain.ErrCAPoolLoad) {
		t.Errorf("missing file: got %v, want ErrCAPoolLoad", err)
	}

	invalid := filepath.Join(dir, "invalid.crt")
	writeFile(t, invalid, []byte("not a certificate"))
	if _, err := New(invalid, slog.New(slog.DiscardHandler)); !errors.Is(err, domain.ErrCAPoolLoad) {
		t.Errorf("invalid file: got %v, want ErrCAPoolLoad", err)
	}
}

func TestBundleTrustsEveryCA(t *testing.T) {
	caV1, caV2 := newCA(t, "v1"), newCA(t, "v2")
	path := filepath.Join(t.TempDir(), "ca.crt")
	writeFile(t, path, append(append([]byte{}, caV1.pem...), caV2.pem...))

	w := newWatcher(t, path)
	for name, ca := range map[string]*testCA{"v1": caV1, "v2": caV2} {
		if err := w.VerifyClientConnection(clientState(ca.leaf(t, x509.ExtKeyUsageClientAuth))); err != nil {
			t.Errorf("client signed by %s rejected: %v", name, err)
		}
	}
}

func TestVerifyClientConnectionRejects(t *testing.T) {
	ca, other := newCA(t, "ca"), newCA(t, "other")
	path := filepath.Join(t.TempDir(), "ca.crt")
	writeFile(t, path, ca.pem)
	w := newWatcher(t, path)

	cases := map[string]tls.ConnectionState{
		"no certificate":    {},
		"unknown CA":        clientState(other.leaf(t, x509.ExtKeyUsageClientAuth)),
		"server-only usage": clientState(ca.leaf(t, x509.ExtKeyUsageServerAuth)),
	}
	for name, state := range cases {
		if err := w.VerifyClientConnection(state); !errors.Is(err, domain.ErrCAPoolVerify) {
			t.Errorf("%s: got %v, want ErrCAPoolVerify", name, err)
		}
	}
}

func TestVerifyServerConnectionChecksHostName(t *testing.T) {
	ca := newCA(t, "ca")
	path := filepath.Join(t.TempDir(), "ca.crt")
	writeFile(t, path, ca.pem)
	w := newWatcher(t, path)

	state := clientState(ca.leaf(t, x509.ExtKeyUsageServerAuth))
	state.ServerName = "127.0.0.1"
	if err := w.VerifyServerConnection(state); err != nil {
		t.Errorf("valid server rejected: %v", err)
	}

	state.ServerName = "other.example"
	if err := w.VerifyServerConnection(state); !errors.Is(err, domain.ErrCAPoolVerify) {
		t.Errorf("wrong host name: got %v, want ErrCAPoolVerify", err)
	}
}

func TestStartReloadsRenewedCA(t *testing.T) {
	caV1, caV2 := newCA(t, "v1"), newCA(t, "v2")
	path := filepath.Join(t.TempDir(), "ca.crt")
	writeFile(t, path, caV1.pem)

	w := newWatcher(t, path)
	startWatcher(t, w)

	clientV1 := clientState(caV1.leaf(t, x509.ExtKeyUsageClientAuth))
	clientV2 := clientState(caV2.leaf(t, x509.ExtKeyUsageClientAuth))
	if err := w.VerifyClientConnection(clientV2); err == nil {
		t.Fatal("client v2 accepted before the renewal")
	}

	writeFile(t, path, caV2.pem)
	eventually(t, func() bool { return w.VerifyClientConnection(clientV2) == nil },
		"client v2 still rejected after the renewal")
	if err := w.VerifyClientConnection(clientV1); err == nil {
		t.Error("client v1 still accepted after the renewal")
	}
}

func TestStartKeepsPoolOnInvalidContent(t *testing.T) {
	ca := newCA(t, "ca")
	path := filepath.Join(t.TempDir(), "ca.crt")
	writeFile(t, path, ca.pem)

	w := newWatcher(t, path)
	startWatcher(t, w)

	writeFile(t, path, []byte("broken"))
	time.Sleep(5 * testInterval)

	if err := w.VerifyClientConnection(clientState(ca.leaf(t, x509.ExtKeyUsageClientAuth))); err != nil {
		t.Errorf("previous pool lost after an invalid update: %v", err)
	}
}

// TestStartFollowsKubeletSymlinkSwap mimics a Secret volume update: ca.crt is a
// link to ..data/ca.crt, and ..data is a link that the kubelet swaps at once.
func TestStartFollowsKubeletSymlinkSwap(t *testing.T) {
	caV1, caV2 := newCA(t, "v1"), newCA(t, "v2")
	dir := t.TempDir()
	for name, ca := range map[string]*testCA{"v1": caV1, "v2": caV2} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o700); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, name, "ca.crt"), ca.pem)
	}
	if err := os.Symlink("v1", filepath.Join(dir, "..data")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ca.crt")
	if err := os.Symlink(filepath.Join("..data", "ca.crt"), path); err != nil {
		t.Fatal(err)
	}

	w := newWatcher(t, path)
	startWatcher(t, w)

	if err := os.Symlink("v2", filepath.Join(dir, "..data_tmp")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "..data_tmp"), filepath.Join(dir, "..data")); err != nil {
		t.Fatal(err)
	}

	clientV2 := clientState(caV2.leaf(t, x509.ExtKeyUsageClientAuth))
	eventually(t, func() bool { return w.VerifyClientConnection(clientV2) == nil },
		"CA v2 not loaded after the symlink swap")
}

// TestMutualTLSAfterRenewal runs real handshakes with the TLS configs used by
// the agent, before and after both CAs and all leaves are renewed.
func TestMutualTLSAfterRenewal(t *testing.T) {
	dir := t.TempDir()
	clientCAPath := filepath.Join(dir, "client-ca.crt")
	serverCAPath := filepath.Join(dir, "server-ca.crt")

	caV1, caV2 := newCA(t, "v1"), newCA(t, "v2")
	writeFile(t, clientCAPath, caV1.pem)
	writeFile(t, serverCAPath, caV1.pem)

	clientCAWatcher := newWatcher(t, clientCAPath)
	serverCAWatcher := newWatcher(t, serverCAPath)
	startWatcher(t, clientCAWatcher)
	startWatcher(t, serverCAWatcher)

	var serverCert, clientCert atomic.Pointer[tls.Certificate]
	storeLeaves := func(ca *testCA) {
		server := ca.leaf(t, x509.ExtKeyUsageServerAuth)
		client := ca.leaf(t, x509.ExtKeyUsageClientAuth)
		serverCert.Store(&server)
		clientCert.Store(&client)
	}
	storeLeaves(caV1)

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return serverCert.Load(), nil
		},
		ClientAuth:       tls.RequireAnyClientCert,
		VerifyConnection: clientCAWatcher.VerifyClientConnection,
	})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		ErrorLog:          log.New(io.Discard, "", 0),
		ReadHeaderTimeout: testTimeout,
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	serverURL := "https://" + listener.Addr().String()

	newClient := func(withCert bool) *http.Client {
		config := &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // Replaced by VerifyConnection.
			VerifyConnection:   serverCAWatcher.VerifyServerConnection,
		}
		if withCert {
			config.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return clientCert.Load(), nil
			}
		}
		return &http.Client{Transport: &http.Transport{TLSClientConfig: config, DisableKeepAlives: true}}
	}
	get := func(client *http.Client) error {
		response, err := client.Get(serverURL)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, response.Body)
		return response.Body.Close()
	}

	client := newClient(true)
	if err := get(client); err != nil {
		t.Fatalf("handshake with v1: %v", err)
	}
	if err := get(newClient(false)); err == nil {
		t.Error("client without certificate accepted")
	}

	// Renewal: new CA files and new leaves, without a restart of the server or client.
	writeFile(t, clientCAPath, caV2.pem)
	writeFile(t, serverCAPath, caV2.pem)
	storeLeaves(caV2)

	eventually(t, func() bool { return get(client) == nil }, "handshake still fails after the renewal")

	stale := caV1.leaf(t, x509.ExtKeyUsageClientAuth)
	clientCert.Store(&stale)
	if err := get(client); err == nil {
		t.Error("client signed by the old CA accepted after the renewal")
	}
}
