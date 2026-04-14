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
	archiveLister service.ArchiveLister
	archiveReader service.ArchiveReader
	archiveLocker service.LockerUnlocker
	rootAPIPath   string
}

func NewDownloadSolutionArchive(
	logger *zerolog.Logger,
	archiveLister service.ArchiveLister,
	archiveReader service.ArchiveReader,
	archiveLocker service.LockerUnlocker,
	rootAPIPath string,
) *DownloadSolutionArchive {
	l := logger.With().Str("use_case", "download_solution_archive").Logger()

	return &DownloadSolutionArchive{
		logger:        &l,
		archiveLister: archiveLister,
		archiveReader: archiveReader,
		archiveLocker: archiveLocker,
		rootAPIPath:   rootAPIPath,
	}
}

func (uc *DownloadSolutionArchive) Execute(
	solutionArchive *domain.SolutionArchive,
) (*domain.SolutionArchiveFile, error) {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Downloading solution archive")

	// To avoid simultaneous downloads and deletion of the same solution archive
	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

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
	file, err := uc.archiveReader.GetFile(solutionArchiveFileName)
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
