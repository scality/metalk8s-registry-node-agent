package service

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

type (
	ExternalSolutionArchiveGetter interface {
		GetExternalSolutionArchive(*domain.SolutionArchive, string) error
	}

	GetExternalSolutionArchiveUseCase interface {
		Execute(solutionArchive *domain.SolutionArchive, url string) error
	}
)
