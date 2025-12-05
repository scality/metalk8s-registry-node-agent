package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	SolutionArchiveUnmounter interface {
		UnmountSolutionArchive(*domain.SolutionArchive) error
	}

	UnmountSolutionArchiveUseCase interface {
		Execute(*domain.SolutionArchive) error
	}
)
