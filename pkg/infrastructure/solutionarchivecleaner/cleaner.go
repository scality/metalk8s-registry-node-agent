package solutionarchivecleaner

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type SolutionArchiveCleaner struct {
	ctx        context.Context
	store      service.StorageProvider
	logger     *zerolog.Logger
	deleteChan chan domain.FileEventDetails
}

func NewSolutionArchiveCleaner(
	ctx context.Context,
	logger *zerolog.Logger,
	store service.StorageProvider,
	deleteChan chan domain.FileEventDetails,
) *SolutionArchiveCleaner {
	l := logger.With().Str("infrastructure", "solution_archive_cleaner").Logger()
	return &SolutionArchiveCleaner{
		ctx:        ctx,
		logger:     &l,
		store:      store,
		deleteChan: deleteChan,
	}
}
