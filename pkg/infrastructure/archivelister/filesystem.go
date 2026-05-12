package archivelister

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type FileSystem struct {
	logger                   *zerolog.Logger
	solutionArchivesLocation string
	interestContentFilter    library.ContentFilter
	store                    service.StorageProvider
}

func NewFileSystem(
	logger *zerolog.Logger,
	solutionArchivesLocation string,
	interestContentFilter library.ContentFilter,
	store service.StorageProvider,
) service.ArchiveLister {
	l := logger.With().
		Str("infrastructure", "archive_lister").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
		interestContentFilter:    interestContentFilter,
		store:                    store,
	}
}

var _ service.ArchiveLister = &FileSystem{}

// ListFiles lists all the files in the location.
func (f *FileSystem) ListFiles() ([]string, error) {
	files, err := library.ListDirContentNames(f.solutionArchivesLocation, f.interestContentFilter)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return files, nil
}

// GetSizeFromFileInfos retrieves the size of a file from the storage backend.
func (f *FileSystem) GetArchiveSize(filename string) (int64, error) {
	return f.store.GetArchiveSize(filename)
}

// GetHashFromFileInfos retrieves the hash of a file from the storage backend.
func (f *FileSystem) GetArchiveHash(filename string) (string, error) {
	return f.store.GetArchiveHash(filename)
}
