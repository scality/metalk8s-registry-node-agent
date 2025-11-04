package archivemounter

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type Storage struct {
	store  service.StorageProvider
	logger *zerolog.Logger
}

func NewStorage(
	store service.StorageProvider,
	logger *zerolog.Logger,
) *Storage {
	l := logger.With().Str("infrastructure", "archive_mounter").Logger()
	return &Storage{
		store:  store,
		logger: &l,
	}
}

var _ service.SolutionArchiveMounter = &Storage{}

func (s *Storage) MountSolutionArchive(solutionArchive *domain.SolutionArchive) error {
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
		err := s.store.MountFile(library.GenSolutionArchiveFileName(solutionArchive),
			library.GenSolutionDirName(solutionArchive))
		if err != nil {
			return errors.Stamp(err)
		}
		return nil
	}

	return errors.From(domain.ErrNotFound).
		WithIdentifier(404000).
		WithDetail("solution archive not found").
		WithProperty("solution_archive_name", solutionArchive.Name).
		WithProperty("solution_archive_version", solutionArchive.Version).
		Throw()
}
