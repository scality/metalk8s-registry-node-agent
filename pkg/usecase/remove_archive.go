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
		return errors.Wrap(err)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		err := uc.archiveRemover.DeleteFile(library.GenSolutionArchiveFileName(solutionArchive))
		if err != nil {
			return errors.Wrap(err)
		}
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive removed")

	return nil
}
