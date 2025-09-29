package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	ArtifactRemover interface {
		RemoveArtifact(*domain.Artifact) error
	}

	RemoveArtifactUseCase interface {
		Execute(*domain.Artifact) error
	}
)
