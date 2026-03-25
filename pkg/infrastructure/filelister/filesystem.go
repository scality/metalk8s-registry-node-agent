package filelister

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/storageprovider"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type FileSystem struct {
	logger                   *zerolog.Logger
	solutionArchivesLocation string
	interestContentFilter    library.ContentFilter
	watchedFileStore         *storageprovider.WatchedFileStore
}

var _ service.FileLister = &FileSystem{}

func NewFileSystem(
	logger *zerolog.Logger,
	solutionArchivesLocation string,
	interestContentFilter library.ContentFilter,
	watchedFileStore *storageprovider.WatchedFileStore,
) *FileSystem {
	l := logger.With().
		Str("infrastructure", "file_lister").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
		interestContentFilter:    interestContentFilter,
		watchedFileStore:         watchedFileStore,
	}
}

func (f *FileSystem) ListFiles() ([]string, error) {
	files, err := library.ListDirContentNames(f.solutionArchivesLocation, f.interestContentFilter)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return files, nil
}

func (f *FileSystem) GetSizeFromFileInfos(fileName string) (int64, error) {
	return f.watchedFileStore.GetSize(fileName)
}
