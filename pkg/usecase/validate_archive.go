package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type ValidateSolutionArchive struct {
	logger *zerolog.Logger

	solutionArchiveValidator service.SolutionArchiveValidator
}

func NewValidateSolutionArchive(
	logger *zerolog.Logger,
	solutionArchiveValidator service.SolutionArchiveValidator,
) *ValidateSolutionArchive {
	l := logger.With().Str("use_case", "validate_solution_archive").Logger()

	return &ValidateSolutionArchive{
		logger:                   &l,
		solutionArchiveValidator: solutionArchiveValidator,
	}
}

func (uc *ValidateSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) (bool, error) {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Validating solution archive")

	isValid, err := uc.solutionArchiveValidator.ValidateSolutionArchive(solutionArchive)
	if err != nil {
		return false, errors.Intercept(err).
			WithDetail("failed to validate solution archive").
			Throw()
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive validation ended")

	return isValid, nil
}
