package service

import (
	"context"
	"io"
)

type (
	ExternalDownloader interface {
		// GetDescription retrieves the size of the external solution archive.
		GetDescription(ctx context.Context, url string) (int64, error)
		// Download returns the HTTP response body wrapped with digest verification.
		// The reader computes a SHA-256 hash on-the-fly via io.TeeReader.
		// Close verifies the computed hash against the Content-Digest trailer.
		Download(ctx context.Context, url string, start int64, end int64, size int64) (io.ReadCloser, error)
	}
)
