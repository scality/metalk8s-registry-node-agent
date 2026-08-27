package externaldownloader

import (
	"bytes"
	"context"
	"crypto"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type HTTP struct {
	logger *slog.Logger
	client *http.Client
}

func NewHTTP(logger *slog.Logger, c *http.Client) *HTTP {
	return &HTTP{
		logger: logger.With(slog.String("infrastructure", "external_downloader")),
		client: c,
	}
}

var _ service.ExternalDownloader = &HTTP{}

func (h *HTTP) DownloadPart(
	ctx context.Context,
	downloadURL string,
	start int64,
	end int64,
	size int64,
) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, errors.Wrap(domain.ErrExternalDownloaderInternal,
			errors.WithIdentifier(126),
			errors.WithDetail("failed to create a new HTTP request"),
			errors.CausedBy(err),
		)
	}
	// According to RFC9110, the header should be bytes=%d-%d, without total size
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(domain.ErrExternalDownloaderInternal,
			errors.WithIdentifier(127),
			errors.WithDetail("failed to send the HTTP request"),
			errors.CausedBy(err),
		)
	}

	if resp.StatusCode != http.StatusPartialContent {
		_ = resp.Body.Close() // nolint: errcheck // Best-effort; body must be closed on error paths.
		return nil, errors.Wrap(domain.ErrExternalDownloaderWrongStatusCode,
			errors.WithIdentifier(128),
			errors.WithDetail("wrong status code received"),
			errors.WithProperty("received_status", resp.StatusCode),
			errors.WithProperty("expected_status", http.StatusPartialContent),
		)
	}

	contentRange := resp.Header.Get("Content-Range")
	if contentRange == "" {
		_ = resp.Body.Close() // nolint: errcheck // Best-effort; body must be closed on error paths.
		return nil, errors.Wrap(domain.ErrExternalDownloaderContentRangeMissing,
			errors.WithIdentifier(129),
			errors.WithDetail("Content-Range header is missing in the response"),
		)
	}

	headerStart, headerEnd, headerSize, err := library.ParseContentRange(contentRange)
	if err != nil {
		_ = resp.Body.Close() // nolint: errcheck // Best-effort; body must be closed on error paths.
		return nil, errors.Wrap(err,
			errors.WithIdentifier(130),
			errors.WithDetail("unexpected error while parsing Content-Range header"),
		)
	}

	if headerStart != start || headerEnd != end || headerSize != size {
		_ = resp.Body.Close() // nolint: errcheck // Best-effort; body must be closed on error paths.
		return nil, errors.Wrap(domain.ErrExternalDownloaderNotConforming,
			errors.WithIdentifier(131),
			errors.WithDetail("received wrong range and size"),
			errors.WithProperty("expected_range", fmt.Sprintf("bytes=%d-%d", start, end)),
			errors.WithProperty("expected_size", size),
			errors.WithProperty("received_range", fmt.Sprintf("bytes=%d-%d", headerStart, headerEnd)),
			errors.WithProperty("received_size", headerSize),
		)
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
		return errors.Wrap(domain.ErrExternalDownloaderContentDigestMissing,
			errors.WithIdentifier(132),
			errors.WithDetail("Content-Digest trailer is missing in the response"),
		)
	}

	expected, err := parseAndDecodeContentDigest(contentDigest)
	if err != nil {
		return errors.Wrap(err)
	}

	computed := r.hasher.Sum(nil)
	if !bytes.Equal(expected, computed) {
		return errors.Wrap(domain.ErrExternalDownloaderInternal,
			errors.WithIdentifier(133),
			errors.WithDetail("the hash of the downloaded part does not match the Content-Digest trailer"),
			errors.WithProperty("expected_hash", expected),
			errors.WithProperty("computed_hash", computed),
		)
	}

	return nil
}

func (h *HTTP) GetDescription(ctx context.Context, downloadURL string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, downloadURL, nil)
	if err != nil {
		return 0, errors.Wrap(domain.ErrExternalDownloaderInternal,
			errors.WithIdentifier(134),
			errors.WithDetail("failed to create a new HTTP request"),
			errors.CausedBy(err),
		)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return 0, errors.Wrap(domain.ErrExternalDownloaderInternal,
			errors.WithIdentifier(135),
			errors.WithDetail("failed to send the HTTP request"),
			errors.CausedBy(err),
		)
	}
	defer resp.Body.Close() // nolint: errcheck // No error check on defer.

	// 2. Check for a successful status code
	if resp.StatusCode != http.StatusOK {
		return 0, errors.Wrap(domain.ErrExternalDownloaderWrongStatusCode,
			errors.WithIdentifier(136),
			errors.WithDetail("wrong status code received"),
			errors.WithProperty("received_status", resp.StatusCode),
			errors.WithProperty("expected_status", http.StatusOK),
		)
	}

	if resp.ContentLength < 0 {
		return 0, errors.Wrap(domain.ErrExternalDownloaderContentLengthMissingOrUnknown,
			errors.WithIdentifier(137),
			errors.WithDetail("Content-Length header is missing or unknown in HEAD response"),
			errors.WithProperty("content_length", resp.ContentLength),
		)
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
		return nil, errors.Wrap(domain.ErrExternalDownloaderInternal,
			errors.WithIdentifier(138),
			errors.WithDetail("Content-Digest header is not in the expected format"),
		)
	}

	decoded, err := base64.StdEncoding.DecodeString(matched[1])
	if err != nil {
		return nil, errors.Wrap(domain.ErrExternalDownloaderInternal,
			errors.WithIdentifier(139),
			errors.WithDetail("failed to decode the Content-Digest header"),
			errors.CausedBy(err),
		)
	}

	return decoded, nil
}
