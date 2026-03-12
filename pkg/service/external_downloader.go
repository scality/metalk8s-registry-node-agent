package service

import (
	"context"
	"io"
)

type (
	ExternalDownloader interface {
		// GetDescription retrieves the size of the external solution archive.
		GetDescription(ctx context.Context, url string) (int64, error)
		// Download returns the HTTP response body; the caller must Close it after reading.
		Download(ctx context.Context, url string, hash string, start int64, end int64, size int64) (io.ReadCloser, error)
	}
)
