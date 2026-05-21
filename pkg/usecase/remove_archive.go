package usecase

import (
	"context"
	"log/slog"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type RemoveSolutionArchive struct {
	logger         *slog.Logger
	archiveLister  service.ArchiveLister
	archiveRemover service.ArchiveRemover
	archiveLocker  service.LockerUnlocker
}

func NewRemoveSolutionArchive(
	logger *slog.Logger,
	archiveLister service.ArchiveLister,
	archiveRemover service.ArchiveRemover,
	archiveLocker service.LockerUnlocker,
) *RemoveSolutionArchive {
	return &RemoveSolutionArchive{
		logger:         logger.With(slog.String("use_case", "remove_solution_archive")),
		archiveLister:  archiveLister,
		archiveRemover: archiveRemover,
		archiveLocker:  archiveLocker,
	}
}

func (uc *RemoveSolutionArchive) Execute(ctx context.Context, solutionArchive *domain.SolutionArchive) error {
	uc.logger.DebugContext(ctx, "Removing solution archive",
		slog.Any("solution_archive", solutionArchive),
	)

	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		err := uc.archiveRemover.DeleteFile(library.GenSolutionArchiveFileName(solutionArchive))
		if err != nil {
			return errors.Stamp(err)
		}
	}

	uc.logger.DebugContext(ctx, "Solution archive removed",
		slog.Any("solution_archive", solutionArchive),
	)

	return nil
}
