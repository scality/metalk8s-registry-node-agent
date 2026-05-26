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
		return errors.Wrap(err,
			errors.WithIdentifier(222),
			errors.WithDetail("error on listing solution archives"),
			errors.WithProperty("usecase", "mount_solution_archive"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchive.Version),
		)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		err := uc.archiveMounter.MountFile(library.GenSolutionArchiveFileName(solutionArchive),
			library.GenSolutionDirName(solutionArchive))
		if err != nil {
			if errors.Is(err, domain.ErrMountSolutionArchiveInvalidISO) {
				return errors.Wrap(err,
					errors.WithIdentifier(223),
					errors.WithDetail("solution archive is not a valid ISO file and is therefore deleted"),
					errors.WithProperty("usecase", "mount_solution_archive"),
					errors.WithProperty("solution_archive_name", solutionArchive.Name),
					errors.WithProperty("solution_archive_version", solutionArchive.Version),
				)
			}

			return errors.Wrap(err,
				errors.WithIdentifier(224),
				errors.WithDetail("unexpected error while mounting the solution archive"),
				errors.WithProperty("usecase", "mount_solution_archive"),
				errors.WithProperty("solution_archive_name", solutionArchive.Name),
				errors.WithProperty("solution_archive_version", solutionArchive.Version),
			)
		}
		uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive mounted")
		return nil
	}

	return errors.Wrap(domain.ErrNotFound,
		errors.WithIdentifier(225),
		errors.WithDetail("solution archive not found"),
		errors.WithProperty("solution_archive_name", solutionArchive.Name),
		errors.WithProperty("solution_archive_version", solutionArchive.Version),
	)
}
