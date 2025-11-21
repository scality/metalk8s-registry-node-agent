package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	SolutionArchiveRemover interface {
		RemoveSolutionArchive(*domain.SolutionArchive) error
	}

	RemoveSolutionArchiveUseCase interface {
		Execute(*domain.SolutionArchive) error
	}
)
