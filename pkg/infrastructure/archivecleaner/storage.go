package archivecleaner

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
	l := logger.With().Str("infrastructure", "archive_cleaner").Logger()
	return &Storage{
		store:  store,
		logger: &l,
	}
}

func (s *Storage) CleanUnusedSolutionArchives(usedSolutionArchives []*domain.SolutionArchive) error {
	s.store.Lock()
	defer s.store.Unlock()

	// List all solution archives in the storage
	fileNames, err := s.store.ListFiles()
	if err != nil {
		return errors.Stamp(err)
	}

	// Create a map of used solution archive filenames for quick lookup
	usedFilenames := make(map[string]struct{})
	for _, solutionArchive := range usedSolutionArchives {
		filename := library.GenSolutionArchiveFileName(solutionArchive)
		usedFilenames[filename] = struct{}{}
	}

	// Delete solution archives that are not in the used list
	for _, filename := range fileNames {
		if _, isUsed := usedFilenames[filename]; !isUsed {
			s.logger.Debug().
				Str("filename", filename).
				Msg("deleting unused solution archive")

			err := s.store.DeleteFile(filename)
			if err != nil {
				s.logger.Error().
					Err(err).
					Str("filename", filename).
					Msg("failed to delete unused solution archive")
				return errors.Stamp(err)
			}

			s.logger.Info().
				Str("filename", filename).
				Msg("deleted unused solution archive")
		}
	}

	return nil
}
