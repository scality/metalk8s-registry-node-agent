// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"context"
	"log/slog"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type CleanArchive struct {
	logger      *slog.Logger
	cleaner     service.ArchiveCleaner
	fileWatcher service.FileWatcher
	deleteChan  chan domain.FileEventDetails
}

func NewCleanArchive(
	logger *slog.Logger,
	cleaner service.ArchiveCleaner,
	fileWatcher service.FileWatcher,
	deleteChan chan domain.FileEventDetails,
) *CleanArchive {
	return &CleanArchive{
		logger:      logger.With(slog.String("use_case", "clean_archive")),
		cleaner:     cleaner,
		fileWatcher: fileWatcher,
		deleteChan:  deleteChan,
	}
}

func (uc *CleanArchive) Execute(ctx context.Context) {
	uc.logger.DebugContext(ctx, "Cleaning archives")
	for eventDetails := range uc.deleteChan {
		switch eventDetails.Origin {
		case domain.SolutionArchivesOrigin:
			err := uc.cleaner.CleanUnusedSolutionArchives(
				ctx,
				eventDetails.FullPathName,
				eventDetails.IsDir,
			)
			if err != nil {
				uc.logger.ErrorContext(ctx, "failed to clean unused solution archive",
					slog.String("path", eventDetails.FullPathName),
					slog.String("origin", "solution_archives"),
					slog.Any("error_message", err),
				)
				uc.fileWatcher.AddWatchFileOrDirectory(eventDetails.FullPathName) // nolint: errcheck // was existing before
			}
		case domain.SolutionsOrigin:
			err := uc.cleaner.CleanUnusedSolutions(
				ctx,
				eventDetails.FullPathName,
				eventDetails.IsDir,
			)
			if err != nil {
				uc.logger.ErrorContext(ctx, "failed to clean unused solution",
					slog.String("path", eventDetails.FullPathName),
					slog.String("origin", "solutions"),
					slog.Any("error_message", err),
				)
				uc.fileWatcher.AddWatchFileOrDirectory(eventDetails.FullPathName) // nolint: errcheck // was existing before
			}
		}
	}
}
