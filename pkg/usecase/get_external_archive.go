package usecase

import (
	"github.com/rs/zerolog"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type GetExternalSolutionArchive struct {
	logger *zerolog.Logger

	externalSolutionArchiveGetter service.ExternalSolutionArchiveGetter
}

func NewGetExternalSolutionArchive(
	logger *zerolog.Logger,
	externalSolutionArchiveGetter service.ExternalSolutionArchiveGetter,
) *GetExternalSolutionArchive {
	l := logger.With().Str("use_case", "get_external_solution_archive").Logger()

	return &GetExternalSolutionArchive{
		logger:                        &l,
		externalSolutionArchiveGetter: externalSolutionArchiveGetter,
	}
}

func (uc *GetExternalSolutionArchive) Execute(solutionArchive *domain.SolutionArchive, url string) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Str("url", url).
		Msg("Getting external solution archive")

	err := uc.externalSolutionArchiveGetter.GetExternalSolutionArchive(solutionArchive, url)
	if err != nil {
		return errors.Intercept(err).
			WithDetail("failed to get external solution archive").
			Throw()
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("External solution archive retrieved")

	return nil
}
