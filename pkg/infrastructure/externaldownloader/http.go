package externaldownloader

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/handler"
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

func (h *HTTP) Download(
	ctx context.Context,
	downloadURL string,
	hash string,
	start int64,
	end int64,
	size int64,
) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("failed to create a new HTTP request").
			CausedBy(err).
			Throw()
	}
	// Add Headers on the request
	req.Header.Set("X-Sha256-checksum", hash)
	// According to RFC9110, the header should be bytes=%d-%d, without total size
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

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

	// Check range and size through the Content-Range header
	contentRange := resp.Header.Get("Content-Range")
	if contentRange == "" {
		_ = resp.Body.Close() // nolint: errcheck // Best-effort; body must be closed on error paths.
		return nil, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("Content-Range header is missing in the response").
			Throw()
	}

	headerStart, headerEnd, headerSize, err := handler.ParseContentRange(contentRange)
	if err != nil {
		_ = resp.Body.Close() // nolint: errcheck // Best-effort; body must be closed on error paths.
		return nil, errors.Stamp(err)
	}

	if headerStart != start || headerEnd != end || headerSize != size {
		_ = resp.Body.Close() // nolint: errcheck // Best-effort; body must be closed on error paths.
		return nil, errors.From(domain.ErrExternalDownloaderNotConforming).
			WithIdentifier(500000).
			WithDetail("received wrong range and size").
			WithProperty("expected_range", fmt.Sprintf("bytes=%d-%d", start, end)).
			WithProperty("expected_size", size).
			WithProperty("received_range", fmt.Sprintf("bytes=%d-%d", headerStart, headerEnd)).
			WithProperty("received_size", headerSize).
			Throw()
	}

	// Stream via io.Copy in the storage layer; do not buffer the chunk in memory.
	return resp.Body, nil
}

func (h *HTTP) GetDescription(ctx context.Context, downloadURL string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, downloadURL, nil)
	if err != nil {
		return 0, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("failed to create a new HTTP request").
			CausedBy(err).
			Throw()
	}
	resp, err := h.client.Do(req)
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
			WithProperty("status", resp.Status).
			Throw()
	}

	return resp.ContentLength, nil
}
