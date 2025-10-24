package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	PartUploader interface {
		UploadPart(*domain.Part) (*domain.ArtifactStatus, error)
	}

	UploadPartUseCase interface {
		Execute(*domain.Part) (*domain.ArtifactStatus, error)
	}
)
