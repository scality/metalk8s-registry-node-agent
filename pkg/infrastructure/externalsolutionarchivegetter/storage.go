package externalsolutionarchivegetter

import (
	"github.com/rs/zerolog"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type Storage struct {
	store              service.StorageProvider
	logger             *zerolog.Logger
	externalDownloader service.ExternalDownloader
}

func NewStorage(
	store service.StorageProvider,
	logger *zerolog.Logger,
	externalDownloader service.ExternalDownloader,
) *Storage {
	l := logger.With().Str("infrastructure", "external_solution_archive_getter").Logger()
	return &Storage{
		store:              store,
		logger:             &l,
		externalDownloader: externalDownloader,
	}
}

var _ service.ExternalSolutionArchiveGetter = &Storage{}

func (s *Storage) GetExternalSolutionArchive(solutionArchive *domain.SolutionArchive, downloadURL string) error {
	s.store.Lock()
	defer s.store.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := s.store.ListFiles()
	if err != nil {
		return errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		return nil
	}

	body, err := s.externalDownloader.Download(downloadURL)
	if err != nil {
		return errors.Stamp(err)
	}

	err = s.store.SaveFile(library.GenSolutionArchiveFileName(solutionArchive),
		body, library.FileSystemDefaultFileMode)
	if err != nil {
		return errors.Stamp(err)
	}

	return nil
}
