package externaldownloader

import (
	"bytes"
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

func (h *HTTP) Download(downloadURL string) (io.Reader, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("failed to create a new HTTP request").
			CausedBy(err).
			Throw()
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("failed to send the HTTP request").
			CausedBy(err).
			Throw()
	}
	defer resp.Body.Close() // nolint: errcheck // No error check on defer.

	// 2. Check for a successful status code
	if resp.StatusCode != http.StatusOK {
		return nil, errors.From(domain.ErrExternalDownloaderNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found").
			WithProperty("status", resp.Status).
			Throw()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("failed to read the response body").
			CausedBy(err).
			Throw()
	}
	return bytes.NewReader(body), nil
}
