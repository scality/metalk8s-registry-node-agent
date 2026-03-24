package service

import "io"

// FileReader provides read-only access to root-level files.
type FileReader interface {
	GetFile(fileName string) (io.ReadCloser, error)
	GetPart(fileName string, start int64, end int64) (io.ReadCloser, error)
}
