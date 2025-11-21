package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	SolutionArchiveValidator interface {
		ValidateSolutionArchive(*domain.SolutionArchive) (bool, error)
	}

	ValidateSolutionArchiveUseCase interface {
		Execute(*domain.SolutionArchive) error
	}
)
