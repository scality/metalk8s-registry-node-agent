package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type ValidateSolutionArchive struct {
	logger *zerolog.Logger
	store  service.StorageProvider
}

func NewValidateSolutionArchive(
	logger *zerolog.Logger,
	store service.StorageProvider,
) *ValidateSolutionArchive {
	l := logger.With().Str("use_case", "validate_solution_archive").Logger()

	return &ValidateSolutionArchive{
		logger: &l,
		store:  store,
	}
}

func (uc *ValidateSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) (bool, error) {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Validating solution archive")

	uc.store.Lock()
	defer uc.store.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.store.ListFiles()
	if err != nil {
		return false, errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchive, fileNames) {
		return false, nil
	}

	hash, err := uc.store.GetHashFromFileInfos(library.GenSolutionArchiveFileName(solutionArchive))
	if err != nil {
		return false, errors.Stamp(err)
	}
	if hash != solutionArchive.Hash {
		err := uc.store.DeleteFile(library.GenSolutionArchiveFileName(solutionArchive))
		if err != nil {
			return false, errors.Stamp(err)
		}
		return false, nil
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive validation ended")

	return true, nil
}
