package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type InitializeSession struct {
	logger *zerolog.Logger

	sessionInitializer service.SessionInitializer
}

func NewInitializeSession(
	logger *zerolog.Logger,
	sessionInitializer service.SessionInitializer,
) *InitializeSession {
	l := logger.With().Str("use_case", "initialize_session").Logger()

	return &InitializeSession{
		logger:             &l,
		sessionInitializer: sessionInitializer,
	}
}

func (uc *InitializeSession) Execute(artifact *domain.Artifact) (*domain.SessionStatus, error) {
	uc.logger.Debug().
		Any("artifact", artifact).
		Msg("Initializing session")

	sessionStatus, err := uc.sessionInitializer.InitializeSession(artifact)
	if err != nil {
		return nil, errors.Intercept(err).
			WithDetail("failed to initialize session").
			Throw()
	}

	uc.logger.Debug().Any("session_status", sessionStatus).Msg("Session initialized")

	return sessionStatus, nil
}
