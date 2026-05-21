package usecase

import (
	"context"
	"log/slog"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type ValidateSolutionArchive struct {
	logger         *slog.Logger
	archiveLister  service.ArchiveLister
	archiveRemover service.ArchiveRemover
	archiveLocker  service.LockerUnlocker
}

func NewValidateSolutionArchive(
	logger *slog.Logger,
	archiveLister service.ArchiveLister,
	archiveRemover service.ArchiveRemover,
	archiveLocker service.LockerUnlocker,
) *ValidateSolutionArchive {
	return &ValidateSolutionArchive{
		logger:         logger.With(slog.String("use_case", "validate_solution_archive")),
		archiveLister:  archiveLister,
		archiveRemover: archiveRemover,
		archiveLocker:  archiveLocker,
	}
}

func (uc *ValidateSolutionArchive) Execute(ctx context.Context, solutionArchive *domain.SolutionArchive) (bool, error) {
	uc.logger.DebugContext(ctx, "Validating solution archive",
		slog.Any("solution_archive", solutionArchive),
	)

	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return false, errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchive, fileNames) {
		return false, nil
	}

	if solutionArchive.Hash != nil {
		hash, err := uc.archiveLister.GetArchiveHash(library.GenSolutionArchiveFileName(solutionArchive))
		if err != nil {
			return false, errors.Stamp(err)
		}
		if hash != *solutionArchive.Hash {
			err := uc.archiveRemover.DeleteFile(library.GenSolutionArchiveFileName(solutionArchive))
			if err != nil {
				return false, errors.Stamp(err)
			}
			return false, nil
		}
	}
	uc.logger.DebugContext(ctx, "Solution archive validation ended",
		slog.Any("solution_archive", solutionArchive),
	)

	return true, nil
}
