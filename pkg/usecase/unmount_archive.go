package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type UnmountSolutionArchive struct {
	logger *zerolog.Logger

	solutionArchiveUnmounter service.SolutionArchiveUnmounter
}

func NewUnmountSolutionArchive(
	logger *zerolog.Logger,
	solutionArchiveUnmounter service.SolutionArchiveUnmounter,
) *UnmountSolutionArchive {
	l := logger.With().Str("use_case", "unmount_solution_archive").Logger()

	return &UnmountSolutionArchive{
		logger:                   &l,
		solutionArchiveUnmounter: solutionArchiveUnmounter,
	}
}

func (uc *UnmountSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Unmounting solution archive")

	err := uc.solutionArchiveUnmounter.UnmountSolutionArchive(solutionArchive)
	if err != nil {
		return errors.Intercept(err).
			WithDetail("failed to unmount solution archive").
			Throw()
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive unmounting ended")

	return nil
}
