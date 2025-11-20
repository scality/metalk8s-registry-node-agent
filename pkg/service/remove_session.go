package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	SessionRemover interface {
		RemoveSession(*domain.SolutionArchive) error
	}

	RemoveSessionUseCase interface {
		Execute(*domain.SolutionArchive) error
	}
)
