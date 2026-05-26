package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type RemoveSolutionArchive struct {
	logger         *zerolog.Logger
	archiveLister  service.ArchiveLister
	archiveRemover service.ArchiveRemover
	archiveLocker  service.LockerUnlocker
}

func NewRemoveSolutionArchive(
	logger *zerolog.Logger,
	archiveLister service.ArchiveLister,
	archiveRemover service.ArchiveRemover,
	archiveLocker service.LockerUnlocker,
) *RemoveSolutionArchive {
	l := logger.With().Str("use_case", "remove_solution_archive").Logger()

	return &RemoveSolutionArchive{
		logger:         &l,
		archiveLister:  archiveLister,
		archiveRemover: archiveRemover,
		archiveLocker:  archiveLocker,
	}
}

func (uc *RemoveSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Removing solution archive")

	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(226),
			errors.WithDetail("error on listing solution archives"),
			errors.WithProperty("usecase", "remove_solution_archive"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchive.Version),
		)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		err := uc.archiveRemover.DeleteFile(library.GenSolutionArchiveFileName(solutionArchive))
		if err != nil {
			return errors.Wrap(err,
				errors.WithIdentifier(227),
				errors.WithDetail("unexpected error while removing the solution archive"),
				errors.WithProperty("usecase", "remove_solution_archive"),
				errors.WithProperty("solution_archive_name", solutionArchive.Name),
				errors.WithProperty("solution_archive_version", solutionArchive.Version),
			)
		}
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive removed")

	return nil
}
