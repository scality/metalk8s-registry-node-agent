package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type RemoveSolutionArchive struct {
	logger *zerolog.Logger

	solutionArchiveRemover service.SolutionArchiveRemover
}

func NewRemoveSolutionArchive(
	logger *zerolog.Logger,
	solutionArchiveRemover service.SolutionArchiveRemover,
) *RemoveSolutionArchive {
	l := logger.With().Str("use_case", "remove_solution_archive").Logger()

	return &RemoveSolutionArchive{
		logger:                 &l,
		solutionArchiveRemover: solutionArchiveRemover,
	}
}

func (uc *RemoveSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Removing solution archive")

	err := uc.solutionArchiveRemover.RemoveSolutionArchive(solutionArchive)
	if err != nil {
		return errors.Intercept(err).
			WithDetail("failed to remove solution archive").
			Throw()
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive removed")

	return nil
}
