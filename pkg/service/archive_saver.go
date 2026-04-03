package service

import (
	"io"
	"os"
)

// ArchiveSaver saves solution archives.
type ArchiveSaver interface {
	// SaveFile saves a file to the root location in the storage. The fileName
	// MUST be unique relative to the root location.
	//
	// When supported by the storage backend, the perm parameter is used to set
	// the file permissions.
	//
	// The caller should close the content reader as soon as possible after
	// this method returns.
	SaveFile(fileName string, content io.Reader, perm os.FileMode) error
}
