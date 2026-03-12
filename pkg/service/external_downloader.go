package service

import "io"

type (
	ExternalDownloader interface {
		GetDescription(url string) (int64, error)
		// Download returns the HTTP response body; the caller must Close it after reading.
		Download(url string, hash string, start int64, end int64, size int64) (io.ReadCloser, error)
	}
)
