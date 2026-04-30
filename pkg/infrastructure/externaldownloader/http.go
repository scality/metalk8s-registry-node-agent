package externaldownloader

import (
	"bytes"
	"context"
	"crypto"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"net/http"
	"regexp"

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

	if resp.StatusCode != http.StatusPartialContent {
		_ = resp.Body.Close() // nolint: errcheck // Best-effort; body must be closed on error paths.
		return nil, errors.From(domain.ErrExternalDownloaderNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found").
			WithProperty("status", resp.StatusCode).
			Throw()
	}

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

	return newDigestVerifyingReader(resp), nil
}

// digestVerifyingReader wraps a response body with io.TeeReader to compute SHA-256
// on-the-fly. On Close, it reads the Content-Digest trailer and verifies integrity.
type digestVerifyingReader struct {
	tee    io.Reader
	body   io.ReadCloser
	hasher hash.Hash
	resp   *http.Response
}

func newDigestVerifyingReader(resp *http.Response) *digestVerifyingReader {
	hasher := crypto.SHA256.New()
	return &digestVerifyingReader{
		tee:    io.TeeReader(resp.Body, hasher),
		body:   resp.Body,
		hasher: hasher,
		resp:   resp,
	}
}

func (r *digestVerifyingReader) Read(p []byte) (int, error) {
	return r.tee.Read(p)
}

func (r *digestVerifyingReader) Close() error {
	defer r.body.Close() // nolint: errcheck // Best-effort close of underlying body.

	contentDigest := r.resp.Trailer.Get("Content-Digest")
	if contentDigest == "" {
		return errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("Content-Digest trailer is missing in the response").
			Throw()
	}

	expected, err := parseAndDecodeContentDigest(contentDigest)
	if err != nil {
		return errors.Stamp(err)
	}

	computed := r.hasher.Sum(nil)
	if !bytes.Equal(expected, computed) {
		return errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("the hash of the downloaded part does not match the Content-Digest trailer").
			WithProperty("expected_hash", expected).
			WithProperty("computed_hash", computed).
			Throw()
	}

	return nil
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

// contentDigestRegexp is used to parse the Content-Digest header, conform to RFC 9530.
var contentDigestRegexp = regexp.MustCompile(
	`^sha-256=:(?P<hash>[A-Za-z0-9+\/=]{44}):$`,
)

// parseAndDecodeContentDigest parses and decodes the Content-Digest header.
func parseAndDecodeContentDigest(contentDigest string) ([]byte, error) {
	matched := contentDigestRegexp.FindStringSubmatch(contentDigest)
	if len(matched) != 2 {
		return nil, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("Content-Digest header is not in the expected format").
			Throw()
	}

	decoded, err := base64.StdEncoding.DecodeString(matched[1])
	if err != nil {
		return nil, errors.From(domain.ErrExternalDownloaderInternal).
			WithIdentifier(500000).
			WithDetail("failed to decode the Content-Digest header").
			CausedBy(err).
			Throw()
	}

	return decoded, nil
}
