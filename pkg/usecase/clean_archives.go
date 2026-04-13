// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type CleanArchives struct {
	logger     *zerolog.Logger
	store      service.StorageProvider
	deleteChan chan domain.FileEventDetails
}

func NewCleanArchives(
	logger *zerolog.Logger,
	store service.StorageProvider,
	deleteChan chan domain.FileEventDetails,
) *CleanArchives {
	l := logger.With().Str("use_case", "clean_archives").Logger()

	return &CleanArchives{
		logger:     &l,
		store:      store,
		deleteChan: deleteChan,
	}
}

func (uc *CleanArchives) Execute() {
	uc.logger.Debug().
		Msg("Cleaning archives")
	for eventDetails := range uc.deleteChan {
		switch eventDetails.Origin {
		case domain.SolutionArchivesOrigin:
			err := uc.store.CleanUnusedSolutionArchives(eventDetails.FullPathName, eventDetails.IsDir)
			if err != nil {
				uc.logger.Error().
					Err(err).
					Str("path", eventDetails.FullPathName).
					Str("origin", "solution_archives").
					Msg("failed to clean unused solution archive")
				uc.store.AddWatchFileOrDirectory(eventDetails.FullPathName) // nolint: errcheck // was existing before
			}
		case domain.SolutionsOrigin:
			err := uc.store.CleanUnusedSolutions(eventDetails.FullPathName, eventDetails.IsDir)
			if err != nil {
				uc.logger.Error().
					Err(err).
					Str("path", eventDetails.FullPathName).
					Str("origin", "solutions").
					Msg("failed to clean unused solution")
				uc.store.AddWatchFileOrDirectory(eventDetails.FullPathName) // nolint: errcheck // was existing before
			}
		}
	}
}
