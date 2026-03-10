// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type DescribeSolutionArchive struct {
	logger *zerolog.Logger

	solutionArchiveDescriber service.SolutionArchiveDescriber
}

func NewDescribeSolutionArchive(
	logger *zerolog.Logger,
	solutionArchiveDescriber service.SolutionArchiveDescriber,
) *DescribeSolutionArchive {
	l := logger.With().Str("use_case", "describe_solution_archive").Logger()

	return &DescribeSolutionArchive{
		logger:                   &l,
		solutionArchiveDescriber: solutionArchiveDescriber,
	}
}

func (uc *DescribeSolutionArchive) Execute(
	solutionArchive *domain.SolutionArchive,
) (int64, error) {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Describing solution archive")

	description, err := uc.solutionArchiveDescriber.DescribeSolutionArchive(solutionArchive)
	if err != nil {
		return 0, errors.Intercept(err).
			WithDetail("failed to describe solution archive").
			Throw()
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive described")

	return description, nil
}
