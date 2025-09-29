package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	SessionRemover interface {
		RemoveSession(*domain.Artifact) error
	}

	RemoveSessionUseCase interface {
		Execute(*domain.Artifact) error
	}
)
