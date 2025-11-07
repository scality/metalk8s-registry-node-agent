package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type CleanUnusedSolutionArchives struct {
	logger *zerolog.Logger

	solutionArchiveCleaner service.SolutionArchiveCleaner
}

func NewCleanUnusedSolutionArchives(
	logger *zerolog.Logger,
	solutionArchiveCleaner service.SolutionArchiveCleaner,
) *CleanUnusedSolutionArchives {
	l := logger.With().Str("use_case", "clean_unused_solution_archives").Logger()

	return &CleanUnusedSolutionArchives{
		logger:                 &l,
		solutionArchiveCleaner: solutionArchiveCleaner,
	}
}

func (uc *CleanUnusedSolutionArchives) Execute(usedSolutionArchives []*domain.SolutionArchive) error {
	uc.logger.Debug().
		Msg("cleaning unused solution archives")

	err := uc.solutionArchiveCleaner.CleanUnusedSolutionArchives(usedSolutionArchives)
	if err != nil {
		return errors.Intercept(err).
			WithDetail("failed to clean unused solution archives").
			Throw()
	}

	uc.logger.Debug().
		Msg("unused solution archives cleaned")

	return nil
}
