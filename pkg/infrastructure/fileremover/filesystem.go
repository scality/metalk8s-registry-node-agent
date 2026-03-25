package fileremover

import (
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/storageprovider"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

var _ service.FileRemover = &FileSystem{}

type FileSystem struct {
	logger                   *zerolog.Logger
	solutionArchivesLocation string
	watchedFileStore         *storageprovider.WatchedFileStore
}

func NewFileSystem(
	logger *zerolog.Logger,
	solutionArchivesLocation string,
	watchedFileStore *storageprovider.WatchedFileStore,
) *FileSystem {
	l := logger.With().
		Str("infrastructure", "file_remover").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
		watchedFileStore:         watchedFileStore,
	}
}

func (f *FileSystem) DeleteFile(fileName string) error {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return errors.Stamp(err)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return errors.Stamp(err)
	}

	if err := library.DeleteFile(filePath); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

func (f *FileSystem) GetHashFromFileInfos(fileName string) (string, error) {
	return f.watchedFileStore.GetHash(fileName)
}
