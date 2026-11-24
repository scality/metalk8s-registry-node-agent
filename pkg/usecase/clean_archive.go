// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"context"
	"log/slog"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

// CleanArchive is the garbage collector that removes archives and mounts
// with no matching NodeSolutionArchive.
type CleanArchive struct {
	logger      *slog.Logger
	cleaner     service.ArchiveCleaner
	fileWatcher service.FileWatcher
	deleteChan  chan domain.FileEventDetails
	disabled    bool
}

// NewCleanArchive returns a CleanArchive. When disabled is true, the
// deletion requests are read but nothing is removed.
func NewCleanArchive(
	logger *slog.Logger,
	cleaner service.ArchiveCleaner,
	fileWatcher service.FileWatcher,
	deleteChan chan domain.FileEventDetails,
	disabled bool,
) *CleanArchive {
	return &CleanArchive{
		logger:      logger.With(slog.String("use_case", "clean_archive")),
		cleaner:     cleaner,
		fileWatcher: fileWatcher,
		deleteChan:  deleteChan,
		disabled:    disabled,
	}
}

// Execute reads deletion requests until the channel is closed. It removes
// the unused archive or mount, unless the garbage collector is disabled.
func (uc *CleanArchive) Execute(ctx context.Context) {
	if uc.disabled {
		uc.logger.InfoContext(ctx, "garbage collector disabled, unused archives are kept")
	} else {
		uc.logger.DebugContext(ctx, "Cleaning archives")
	}
	for eventDetails := range uc.deleteChan {
		if uc.disabled {
			uc.logger.InfoContext(ctx, "garbage collector disabled, skipping clean",
				slog.String("path", eventDetails.FullPathName),
			)
			continue
		}
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
					slog.Any("error", err),
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
					slog.Any("error", err),
				)
				uc.fileWatcher.AddWatchFileOrDirectory(eventDetails.FullPathName) // nolint: errcheck // was existing before
			}
		}
	}
}
