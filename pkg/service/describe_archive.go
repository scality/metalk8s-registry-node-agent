package service

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

type (
	SolutionArchiveDescriber interface {
		DescribeSolutionArchive(*domain.SolutionArchive) (int64, error)
	}

	DescribeSolutionArchiveUseCase interface {
		Execute(*domain.SolutionArchive) (int64, error)
	}
)
