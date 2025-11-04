package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type MountSolutionArchive struct {
	logger *zerolog.Logger

	solutionArchiveMounter service.SolutionArchiveMounter
}

func NewMountSolutionArchive(
	logger *zerolog.Logger,
	solutionArchiveMounter service.SolutionArchiveMounter,
) *MountSolutionArchive {
	l := logger.With().Str("use_case", "mount_solution_archive").Logger()

	return &MountSolutionArchive{
		logger:                 &l,
		solutionArchiveMounter: solutionArchiveMounter,
	}
}

func (uc *MountSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Mounting solution archive")

	err := uc.solutionArchiveMounter.MountSolutionArchive(solutionArchive)
	if err != nil {
		return errors.Intercept(err).
			WithDetail("failed to mount solution archive").
			Throw()
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive mounted")

	return nil
}
