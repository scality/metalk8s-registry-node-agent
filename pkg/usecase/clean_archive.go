// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type CleanArchive struct {
	logger      *zerolog.Logger
	cleaner     service.ArchiveCleaner
	fileWatcher service.FileWatcher
	deleteChan  chan domain.FileEventDetails
}

func NewCleanArchive(
	logger *zerolog.Logger,
	cleaner service.ArchiveCleaner,
	fileWatcher service.FileWatcher,
	deleteChan chan domain.FileEventDetails,
) *CleanArchive {
	l := logger.With().Str("use_case", "clean_archive").Logger()

	return &CleanArchive{
		logger:      &l,
		cleaner:     cleaner,
		fileWatcher: fileWatcher,
		deleteChan:  deleteChan,
	}
}

func (uc *CleanArchive) Execute() {
	uc.logger.Debug().
		Msg("Cleaning archives")
	for eventDetails := range uc.deleteChan {
		switch eventDetails.Origin {
		case domain.SolutionArchivesOrigin:
			err := uc.cleaner.CleanUnusedSolutionArchives(
				eventDetails.FullPathName,
				eventDetails.IsDir,
			)
			if err != nil {
				uc.logger.Error().
					Err(err).
					Str("path", eventDetails.FullPathName).
					Str("origin", "solution_archives").
					Msg("failed to clean unused solution archive")
				uc.fileWatcher.AddWatchFileOrDirectory(eventDetails.FullPathName) // nolint: errcheck // was existing before
			}
		case domain.SolutionsOrigin:
			err := uc.cleaner.CleanUnusedSolutions(
				eventDetails.FullPathName,
				eventDetails.IsDir,
			)
			if err != nil {
				uc.logger.Error().
					Err(err).
					Str("path", eventDetails.FullPathName).
					Str("origin", "solutions").
					Msg("failed to clean unused solution")
				uc.fileWatcher.AddWatchFileOrDirectory(eventDetails.FullPathName) // nolint: errcheck // was existing before
			}
		}
	}
}
