package archivevalidator

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
	l := logger.With().Str("infrastructure", "archivevalidator").Logger()
	return &Storage{
		store:  store,
		logger: &l,
	}
}

func (s *Storage) ValidateSolutionArchive(solutionArchive *domain.SolutionArchive) (bool, error) {
	s.store.Lock()
	defer s.store.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := s.store.ListFiles()
	if err != nil {
		return false, errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchive, fileNames) {
		return false, nil
	}

	hash, err := s.store.GetHashFromFileInfos(library.GenSolutionArchiveFileName(solutionArchive))
	if err != nil {
		return false, errors.Stamp(err)
	}
	if hash != solutionArchive.Hash {
		err := s.store.DeleteFile(library.GenSolutionArchiveFileName(solutionArchive))
		if err != nil {
			return false, errors.Stamp(err)
		}
		return false, nil
	}

	return true, nil
}
