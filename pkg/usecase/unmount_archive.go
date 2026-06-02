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

	properties := solutionArchive.GetErrorProperties(
		"unmount_solution_archive",
		"",
	)

	err := uc.archiveMounter.UnmountFile(library.GenSolutionDirName(solutionArchive))
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(231),
			errors.WithDetail("unexpected error while unmounting the solution archive"),
			errors.WithProperties(properties),
		)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive unmounting ended")

	return nil
}
