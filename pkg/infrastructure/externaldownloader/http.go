package externaldownloader

import (
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type HTTP struct {
	logger *zerolog.Logger
	client *http.Client
}

func NewHTTP(logger *zerolog.Logger, c *http.Client) *HTTP {
	l := logger.With().Str("infrastructure", "external_downloader").Logger()
	return &HTTP{
		logger: &l,
		client: c,
	}
}

var _ service.ExternalDownloader = &HTTP{}

func (h *HTTP) Download(downloadURL string, hash string, start int64, end int64, size int64) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("failed to create a new HTTP request").
			CausedBy(err).
			Throw()
	}
	// Add Headers on the request
	req.Header.Set("X-Sha256-checksum", hash)
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("failed to send the HTTP request").
			CausedBy(err).
			Throw()
	}

	// 2. Check for a successful status code
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close() // nolint: errcheck // Best-effort; body must be closed on error paths.
		return nil, errors.From(domain.ErrExternalDownloaderNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found").
			WithProperty("status", resp.Status).
			Throw()
	}

	// Stream via io.Copy in the storage layer; do not buffer the chunk in memory.
	return resp.Body, nil
}

func (h *HTTP) GetDescription(downloadURL string) (int64, error) {
	resp, err := h.client.Head(downloadURL)
	if err != nil {
		return 0, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("failed to send the HTTP request").
			CausedBy(err).
			Throw()
	}
	defer resp.Body.Close() // nolint: errcheck // No error check on defer.

	// 2. Check for a successful status code
	if resp.StatusCode != http.StatusOK {
		return 0, errors.From(domain.ErrExternalDownloaderNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found").
			WithProperty("status", resp.Status).
			Throw()
	}

	if resp.ContentLength < 0 {
		return 0, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("Content-Length header is missing or unknown in HEAD response").
			Throw()
	}

	return resp.ContentLength, nil
}
