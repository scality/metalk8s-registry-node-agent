package archiveunmounter

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
	l := logger.With().Str("infrastructure", "archive_unmounter").Logger()
	return &Storage{
		store:  store,
		logger: &l,
	}
}

var _ service.SolutionArchiveUnmounter = &Storage{}

func (s *Storage) UnmountSolutionArchive(solutionArchive *domain.SolutionArchive) error {
	s.store.Lock()
	defer s.store.Unlock()

	err := s.store.UnmountFile(library.GenSolutionDirName(solutionArchive))
	if err != nil {
		return errors.Stamp(err)
	}

	return nil
}
