package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

type (
	PartUploader interface {
		UploadPart(*domain.Part) (*domain.SolutionArchiveStatus, error)
	}

	UploadPartUseCase interface {
		Execute(*domain.Part) (*domain.SolutionArchiveStatus, error)
	}
)
