package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	SolutionArchiveCleaner interface {
		CleanUnusedSolutionArchives([]*domain.SolutionArchive) error
	}

	CleanUnusedSolutionArchivesUseCase interface {
		Execute([]*domain.SolutionArchive) error
	}
)
