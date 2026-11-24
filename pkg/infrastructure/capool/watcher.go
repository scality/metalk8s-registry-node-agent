// Package capool keeps an x509 CA pool in sync with a PEM bundle file on disk.
package capool

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

// defaultWatchInterval is the delay between two reads of the CA bundle file.
const defaultWatchInterval = 10 * time.Second

// Watcher holds the CA pool built from a PEM bundle file and reloads it when
// the file content changes, so a renewed CA is trusted without a restart.
type Watcher struct {
	path     string
	interval time.Duration
	logger   *slog.Logger

	pool atomic.Pointer[x509.CertPool]

	// mu protects lastPEM.
	mu      sync.Mutex
	lastPEM []byte
}

// New reads the CA bundle at path and returns a Watcher that serves it.
// It fails when the file cannot be read or holds no valid certificate.
func New(path string, logger *slog.Logger) (*Watcher, error) {
	w := &Watcher{
		path:     path,
		interval: defaultWatchInterval,
		logger:   logger,
	}

	if _, err := w.reload(); err != nil {
		return nil, err
	}

	return w, nil
}

// WithInterval sets the delay between two reads of the CA bundle file.
func (w *Watcher) WithInterval(interval time.Duration) *Watcher {
	w.interval = interval
	return w
}

// Pool returns the CA pool built from the last valid content of the file.
func (w *Watcher) Pool() *x509.CertPool {
	return w.pool.Load()
}

// Start reads the CA bundle file at each interval until ctx is done.
// An invalid content is logged and the previous pool is kept.
func (w *Watcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.InfoContext(ctx, "starting CA pool watcher",
		slog.String("path", w.path),
		slog.Duration("interval", w.interval),
	)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			updated, err := w.reload()
			if err != nil {
				w.logger.ErrorContext(ctx, "failed to reload CA pool, keeping the previous one",
					slog.String("path", w.path),
					slog.Any("error", err),
				)
				continue
			}
			if updated {
				w.logger.InfoContext(ctx, "CA pool updated", slog.String("path", w.path))
			}
		}
	}
}

// NeedLeaderElection returns false: every replica must watch its own files.
func (w *Watcher) NeedLeaderElection() bool {
	return false
}

// VerifyClientConnection checks the client certificate of a TLS connection
// against the current pool. It is meant for tls.Config.VerifyConnection.
func (w *Watcher) VerifyClientConnection(state tls.ConnectionState) error {
	return w.verify(state, x509.VerifyOptions{
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
}

// VerifyServerConnection checks the server certificate and host name of a TLS
// connection against the current pool. It is meant for tls.Config.VerifyConnection.
func (w *Watcher) VerifyServerConnection(state tls.ConnectionState) error {
	return w.verify(state, x509.VerifyOptions{
		DNSName:   state.ServerName,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
}

// verify checks the peer certificate chain of state with opts and the current pool.
func (w *Watcher) verify(state tls.ConnectionState, opts x509.VerifyOptions) error {
	if len(state.PeerCertificates) == 0 {
		return errors.Wrap(domain.ErrCAPoolVerify,
			errors.WithIdentifier(255),
			errors.WithDetail("peer sent no certificate"),
		)
	}

	opts.Roots = w.Pool()
	opts.Intermediates = x509.NewCertPool()
	for _, cert := range state.PeerCertificates[1:] {
		opts.Intermediates.AddCert(cert)
	}

	if _, err := state.PeerCertificates[0].Verify(opts); err != nil {
		return errors.Wrap(domain.ErrCAPoolVerify,
			errors.WithIdentifier(256),
			errors.WithDetail("peer certificate is not trusted"),
			errors.WithProperty("subject", state.PeerCertificates[0].Subject.String()),
			errors.CausedBy(err),
		)
	}

	return nil
}

// reload reads the file and replaces the pool when the content changed.
// It returns true when the pool was replaced.
func (w *Watcher) reload() (bool, error) {
	content, err := os.ReadFile(w.path)
	if err != nil {
		return false, errors.Wrap(domain.ErrCAPoolLoad,
			errors.WithIdentifier(257),
			errors.WithDetail("failed to read CA bundle file"),
			errors.WithProperty("path", w.path),
			errors.CausedBy(err),
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.lastPEM != nil && bytes.Equal(w.lastPEM, content) {
		return false, nil
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(content) {
		return false, errors.Wrap(domain.ErrCAPoolLoad,
			errors.WithIdentifier(258),
			errors.WithDetail("CA bundle file holds no valid certificate"),
			errors.WithProperty("path", w.path),
		)
	}

	w.pool.Store(pool)
	w.lastPEM = content

	return true, nil
}
