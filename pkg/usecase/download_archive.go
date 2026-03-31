// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type DownloadSolutionArchive struct {
	logger        *zerolog.Logger
	store         service.StorageProvider
	archiveLister service.ArchiveLister
	rootAPIPath   string
}

func NewDownloadSolutionArchive(
	logger *zerolog.Logger,
	store service.StorageProvider,
	archiveLister service.ArchiveLister,
	rootAPIPath string,
) *DownloadSolutionArchive {
	l := logger.With().Str("use_case", "download_solution_archive").Logger()

	return &DownloadSolutionArchive{
		logger:        &l,
		store:         store,
		archiveLister: archiveLister,
		rootAPIPath:   rootAPIPath,
	}
}

func (uc *DownloadSolutionArchive) Execute(
	solutionArchive *domain.SolutionArchive,
) (*domain.SolutionArchiveFile, error) {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Downloading solution archive")

	uc.store.Lock()
	defer uc.store.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return nil, errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchive, fileNames) {
		return nil, errors.From(domain.ErrNotFound).
			WithDetail("solution archive not found").
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, solutionArchive.Name)).
			WithProperty("solution_archive_name", solutionArchive.Name).
			WithProperty("solution_archive_version", solutionArchive.Version).
			Throw()
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(solutionArchive)
	file, err := uc.store.GetFile(solutionArchiveFileName)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	size, err := uc.archiveLister.GetArchiveSize(solutionArchiveFileName)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive downloaded")

	return &domain.SolutionArchiveFile{
		File: file,
		Size: size,
	}, nil
}
