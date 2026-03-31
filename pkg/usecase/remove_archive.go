package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type RemoveSolutionArchive struct {
	logger         *zerolog.Logger
	store          service.StorageProvider
	archiveLister  service.ArchiveLister
	archiveRemover service.ArchiveRemover
}

func NewRemoveSolutionArchive(
	logger *zerolog.Logger,
	store service.StorageProvider,
	archiveLister service.ArchiveLister,
	archiveRemover service.ArchiveRemover,
) *RemoveSolutionArchive {
	l := logger.With().Str("use_case", "remove_solution_archive").Logger()

	return &RemoveSolutionArchive{
		logger:         &l,
		store:          store,
		archiveLister:  archiveLister,
		archiveRemover: archiveRemover,
	}
}

func (uc *RemoveSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Removing solution archive")

	uc.store.Lock()
	defer uc.store.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		err := uc.archiveRemover.DeleteFile(library.GenSolutionArchiveFileName(solutionArchive))
		if err != nil {
			return errors.Stamp(err)
		}
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive removed")

	return nil
}
