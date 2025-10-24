package library

import (
	"io/fs"
	"os"
)

type dirEntry struct {
	os.FileInfo
}

var _ fs.DirEntry = dirEntry{}

func (de dirEntry) Info() (fs.FileInfo, error) {
	return de, nil
}

func (de dirEntry) Type() fs.FileMode {
	return de.Mode()
}
