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
	solutionArchivePart *domain.Part,
) (*domain.SolutionArchiveFile, error) {
	uc.logger.Debug().
		Any("solution_archive_part", solutionArchivePart).
		Msg("Downloading solution archive chunk")

	// To avoid simultaneous downloads and deletion of the same solution archive
	uc.archiveLocker.RLock(solutionArchivePart.SolutionArchive)
	defer uc.archiveLocker.RUnlock(solutionArchivePart.SolutionArchive)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return nil, errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchivePart.SolutionArchive, fileNames) {
		return nil, errors.From(domain.ErrNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found").
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, solutionArchivePart.SolutionArchive.Name)).
			WithProperty("solution_archive_name", solutionArchivePart.SolutionArchive.Name).
			WithProperty("solution_archive_version", solutionArchivePart.SolutionArchive.Version).
			Throw()
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(solutionArchivePart.SolutionArchive)
	fileSize, err := uc.archiveLister.GetArchiveSize(solutionArchiveFileName)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	start := solutionArchivePart.Meta.Start
	end := solutionArchivePart.Meta.End
	if start >= fileSize || end >= fileSize {
		return nil, errors.From(domain.ErrHandlerInvalidRequestHeaderFormat).
			WithIdentifier(400006).
			WithDetail("requested byte range exceeds the solution archive size").
			WithProperty("range_start", start).
			WithProperty("range_end", end).
			WithProperty("file_size_bytes", fileSize).
			Throw()
	}

	// Retrieve the requested part of the solution archive from the storage
	file, err := uc.archiveReader.GetPart(
		solutionArchiveFileName,
		solutionArchivePart.Meta.Start,
		solutionArchivePart.Meta.End,
	)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	uc.logger.Debug().Any("solution_archive_part", solutionArchivePart).Msg("Solution archive chunk downloaded")

	return &domain.SolutionArchiveFile{
		File:          file,
		Size:          fileSize,
		ContentLength: solutionArchivePart.Meta.Size(),
	}, nil
}
