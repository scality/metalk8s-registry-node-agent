package archivedescriber

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type Storage struct {
	store       service.StorageProvider
	logger      *zerolog.Logger
	rootAPIPath string
}

func NewStorage(
	store service.StorageProvider,
	logger *zerolog.Logger,
	rootAPIPath string,
) *Storage {
	l := logger.With().Str("infrastructure", "archive_describer").Logger()
	return &Storage{
		store:       store,
		logger:      &l,
		rootAPIPath: rootAPIPath,
	}
}

var _ service.SolutionArchiveDescriber = &Storage{}

func (s *Storage) DescribeSolutionArchive(
	solutionArchive *domain.SolutionArchive,
) (int64, error) {
	s.store.Lock()
	defer s.store.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := s.store.ListFiles()
	if err != nil {
		return 0, errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchive, fileNames) {
		return 0, errors.From(domain.ErrNotFound).
			WithDetail("solution archive not found").
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", s.rootAPIPath, solutionArchive.Name)).
			WithProperty("solution_archive_name", solutionArchive.Name).
			WithProperty("solution_archive_version", solutionArchive.Version).
			Throw()
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(solutionArchive)
	size, err := s.store.GetSizeFromFileInfos(solutionArchiveFileName)
	if err != nil {
		return 0, errors.Stamp(err)
	}

	return size, nil
}
