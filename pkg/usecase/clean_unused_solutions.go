package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type CleanUnusedSolutions struct {
	logger *zerolog.Logger

	solutionCleaner service.SolutionCleaner
}

func NewCleanUnusedSolutions(
	logger *zerolog.Logger,
	solutionCleaner service.SolutionCleaner,
) *CleanUnusedSolutions {
	l := logger.With().Str("use_case", "clean_unused_solutions").Logger()

	return &CleanUnusedSolutions{
		logger:          &l,
		solutionCleaner: solutionCleaner,
	}
}

func (uc *CleanUnusedSolutions) Execute(usedSolutionArchives []*domain.SolutionArchive) error {
	uc.logger.Debug().
		Msg("cleaning unused solutions")

	err := uc.solutionCleaner.CleanUnusedSolutions(usedSolutionArchives)
	if err != nil {
		return errors.Intercept(err).
			WithDetail("failed to clean unused solutions").
			Throw()
	}

	uc.logger.Debug().
		Msg("unused solutions cleaned")

	return nil
}
