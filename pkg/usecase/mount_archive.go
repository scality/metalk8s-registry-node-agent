package usecase

import (
	"context"
	"log/slog"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type MountSolutionArchive struct {
	logger         *slog.Logger
	archiveMounter service.ArchiveMounter
	archiveLister  service.ArchiveLister
	archiveLocker  service.LockerUnlocker
}

func NewMountSolutionArchive(
	logger *slog.Logger,
	archiveMounter service.ArchiveMounter,
	archiveLister service.ArchiveLister,
	archiveLocker service.LockerUnlocker,
) *MountSolutionArchive {
	return &MountSolutionArchive{
		logger:         logger.With(slog.String("use_case", "mount_solution_archive")),
		archiveMounter: archiveMounter,
		archiveLister:  archiveLister,
		archiveLocker:  archiveLocker,
	}
}

func (uc *MountSolutionArchive) Execute(ctx context.Context, solutionArchive *domain.SolutionArchive) error {
	uc.logger.DebugContext(ctx, "Mounting solution archive",
		slog.Any("solution_archive", solutionArchive),
	)

	// To avoid simultaneous mounts and unmounts of the same solution archive
	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

	properties := solutionArchive.GetErrorProperties(
		"mount_solution_archive",
		"",
	)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(222),
			errors.WithDetail("error on listing solution archives"),
			errors.WithProperties(properties),
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
					errors.WithProperties(properties),
				)
			}

			return errors.Wrap(err,
				errors.WithIdentifier(224),
				errors.WithDetail("unexpected error while mounting the solution archive"),
				errors.WithProperties(properties),
			)
		}
		uc.logger.DebugContext(ctx, "Solution archive mounted",
			slog.Any("solution_archive", solutionArchive),
		)
		return nil
	}

	return errors.Wrap(domain.ErrNotFound,
		errors.WithIdentifier(225),
		errors.WithDetail("solution archive not found"),
		errors.WithProperties(properties),
	)
}
