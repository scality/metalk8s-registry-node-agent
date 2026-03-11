package service

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

type (
	SolutionArchiveDownloader interface {
		DownloadSolutionArchive(*domain.Part) (*domain.SolutionArchiveFile, error)
	}

	DownloadSolutionArchiveUseCase interface {
		Execute(*domain.SolutionArchive) error
	}
)
