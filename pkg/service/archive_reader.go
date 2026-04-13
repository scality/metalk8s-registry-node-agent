package service

import "io"

// ArchiveReader provides read-only access to root-level files.
type ArchiveReader interface {
	// GetFile retrieves the content of a file from the root location in the
	// storage based on its fileName.
	//
	// The caller should close the content reader as early as possible.
	GetFile(fileName string) (io.ReadCloser, error)
}
