package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type ValidateSolutionArchive struct {
	logger      *zerolog.Logger
	fileLister  service.FileLister
	fileRemover service.FileRemover
}

func NewValidateSolutionArchive(
	logger *zerolog.Logger,
	fileLister service.FileLister,
	fileRemover service.FileRemover,
) *ValidateSolutionArchive {
	l := logger.With().Str("use_case", "validate_solution_archive").Logger()

	return &ValidateSolutionArchive{
		logger:      &l,
		fileLister:  fileLister,
		fileRemover: fileRemover,
	}
}

func (uc *ValidateSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) (bool, error) {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Validating solution archive")

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.fileLister.ListFiles()
	if err != nil {
		return false, errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchive, fileNames) {
		return false, nil
	}

	hash, err := uc.fileRemover.GetHashFromFileInfos(library.GenSolutionArchiveFileName(solutionArchive))
	if err != nil {
		return false, errors.Stamp(err)
	}
	if hash != solutionArchive.Hash {
		err := uc.fileRemover.DeleteFile(library.GenSolutionArchiveFileName(solutionArchive))
		if err != nil {
			return false, errors.Stamp(err)
		}
		return false, nil
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive validation ended")

	return true, nil
}
