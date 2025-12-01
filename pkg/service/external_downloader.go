package service

import "io"

type (
	ExternalDownloader interface {
		Download(url string) (io.Reader, error)
	}
)
