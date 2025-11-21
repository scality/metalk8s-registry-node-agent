package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	SessionInitializer interface {
		InitializeSession(solutionArchive *domain.SolutionArchive) (*domain.SessionStatus, error)
	}

	InitializeSessionUseCase interface {
		Execute(*domain.SolutionArchive) (*domain.SessionStatus, error)
	}
)
