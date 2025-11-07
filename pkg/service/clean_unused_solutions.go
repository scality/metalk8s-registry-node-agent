package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	SolutionCleaner interface {
		CleanUnusedSolutions([]*domain.SolutionArchive) error
	}

	CleanUnusedSolutionsUseCase interface {
		Execute([]*domain.SolutionArchive) error
	}
)
