package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	SessionInitializer interface {
		InitializeSession(artifact *domain.Artifact) (*domain.SessionStatus, error)
	}

	InitializeSessionUseCase interface {
		Execute(*domain.Artifact) (*domain.SessionStatus, error)
	}
)
