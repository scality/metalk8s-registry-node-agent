package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type MountSolutionArchive struct {
	logger         *zerolog.Logger
	archiveMounter service.ArchiveMounter
	archiveLister  service.ArchiveLister
	archiveLocker  service.LockerUnlocker
}

func NewMountSolutionArchive(
	logger *zerolog.Logger,
	archiveMounter service.ArchiveMounter,
	archiveLister service.ArchiveLister,
	archiveLocker service.LockerUnlocker,
) *MountSolutionArchive {
	l := logger.With().Str("use_case", "mount_solution_archive").Logger()

	return &MountSolutionArchive{
		logger:         &l,
		archiveMounter: archiveMounter,
		archiveLister:  archiveLister,
		archiveLocker:  archiveLocker,
	}
}

func (uc *MountSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Mounting solution archive")

	// To avoid simultaneous mounts and unmounts of the same solution archive
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
		err := uc.archiveMounter.MountFile(library.GenSolutionArchiveFileName(solutionArchive),
			library.GenSolutionDirName(solutionArchive))
		if err != nil {
			if errors.Is(err, domain.ErrMountSolutionArchiveInvalidISO) {
				return errors.Wrap(err,
					errors.WithDetail("file deleted"),
				)
			}

			return errors.Wrap(err,
				errors.WithDetail("failed to mount solution archive"),
			)
		}
		uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive mounted")
		return nil
	}

	return errors.Wrap(domain.ErrNotFound,
		errors.WithIdentifier(404000),
		errors.WithDetail("solution archive not found"),
		errors.WithProperty("solution_archive_name", solutionArchive.Name),
		errors.WithProperty("solution_archive_version", solutionArchive.Version),
	)
}
