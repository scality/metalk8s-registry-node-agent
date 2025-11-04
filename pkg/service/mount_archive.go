package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	SolutionArchiveMounter interface {
		MountSolutionArchive(*domain.SolutionArchive) error
	}

	MountSolutionArchiveUseCase interface {
		Execute(*domain.SolutionArchive) error
	}
)
