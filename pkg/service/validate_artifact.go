package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	ArtifactValidator interface {
		ValidateArtifact(*domain.Artifact) (bool, error)
	}

	ValidateArtifactUseCase interface {
		Execute(*domain.Artifact) error
	}
)
