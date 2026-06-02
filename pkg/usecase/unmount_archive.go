package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type UnmountSolutionArchive struct {
	logger         *zerolog.Logger
	archiveMounter service.ArchiveMounter
	archiveLocker  service.LockerUnlocker
}

func NewUnmountSolutionArchive(
	logger *zerolog.Logger,
	archiveMounter service.ArchiveMounter,
	archiveLocker service.LockerUnlocker,
) *UnmountSolutionArchive {
	l := logger.With().Str("use_case", "unmount_solution_archive").Logger()

	return &UnmountSolutionArchive{
		logger:         &l,
		archiveMounter: archiveMounter,
		archiveLocker:  archiveLocker,
	}
}

func (uc *UnmountSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Unmounting solution archive")

	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

	err := uc.archiveMounter.UnmountFile(library.GenSolutionDirName(solutionArchive))
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(231),
			errors.WithDetail("unexpected error while unmounting the solution archive"),
			errors.WithProperty("usecase", "unmount_solution_archive"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchive.Version),
		)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive unmounting ended")

	return nil
}
