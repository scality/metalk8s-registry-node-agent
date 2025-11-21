package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type RemoveSession struct {
	logger *zerolog.Logger

	sessionRemover service.SessionRemover
}

func NewRemoveSession(
	logger *zerolog.Logger,
	sessionRemover service.SessionRemover,
) *RemoveSession {
	l := logger.With().Str("use_case", "remove_session").Logger()

	return &RemoveSession{
		logger:         &l,
		sessionRemover: sessionRemover,
	}
}

func (uc *RemoveSession) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Removing session")

	err := uc.sessionRemover.RemoveSession(solutionArchive)
	if err != nil {
		return errors.Intercept(err).
			WithDetail("failed to remove session").
			Throw()
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Session removed")

	return nil
}
