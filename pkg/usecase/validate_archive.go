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
	metrics        service.MetricsRecorder
}

func NewValidateSolutionArchive(
	logger *slog.Logger,
	archiveLister service.ArchiveLister,
	archiveRemover service.ArchiveRemover,
	archiveLocker service.LockerUnlocker,
	metrics service.MetricsRecorder,
) *ValidateSolutionArchive {
	return &ValidateSolutionArchive{
		logger:         logger.With(slog.String("use_case", "validate_solution_archive")),
		archiveLister:  archiveLister,
		archiveRemover: archiveRemover,
		archiveLocker:  archiveLocker,
		metrics:        metrics,
	}
}

func (uc *ValidateSolutionArchive) Execute(ctx context.Context, solutionArchive *domain.SolutionArchive) (bool, error) {
	uc.logger.DebugContext(ctx, "Validating solution archive",
		slog.Any("solution_archive", solutionArchive),
	)

	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

	properties := solutionArchive.GetErrorProperties(
		"validate_solution_archive",
		"",
	)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return false, errors.Wrap(err,
			errors.WithIdentifier(239),
			errors.WithDetail("error on listing solution archives"),
			errors.WithProperties(properties),
		)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchive, fileNames) {
		return false, nil
	}

	if solutionArchive.Hash != nil {
		hash, err := uc.archiveLister.GetArchiveHash(library.GenSolutionArchiveFileName(solutionArchive))
		if err != nil {
			return false, errors.Wrap(err,
				errors.WithIdentifier(240),
				errors.WithDetail("error on getting the archive hash"),
				errors.WithProperties(properties),
			)
		}
		if hash != *solutionArchive.Hash {
			uc.metrics.IncIntegrityFailure(solutionArchive, domain.IntegrityStageArchiveChecksum)
			err := uc.archiveRemover.DeleteFile(library.GenSolutionArchiveFileName(solutionArchive))
			if err != nil {
				return false, errors.Wrap(err,
					errors.WithIdentifier(241),
					errors.WithDetail("error on deleting the solution archive"),
					errors.WithProperties(properties),
				)
			}
			return false, nil
		}
	}
	uc.logger.DebugContext(ctx, "Solution archive validation ended",
		slog.Any("solution_archive", solutionArchive),
	)

	return true, nil
}
