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
		return nil, errors.Wrap(err,
			errors.WithIdentifier(199),
			errors.WithDetail("error on listing solution archives"),
			errors.WithProperty("usecase", "download_solution_archive"),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchivePart.SolutionArchive.Name,
				solutionArchivePart.SolutionArchive.Version,
			)),
		)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchivePart.SolutionArchive, fileNames) {
		return nil, errors.Wrap(domain.ErrNotFound,
			errors.WithIdentifier(200),
			errors.WithDetail("solution archive not found"),
			errors.WithProperty("usecase", "download_solution_archive"),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchivePart.SolutionArchive.Name,
				solutionArchivePart.SolutionArchive.Version,
			)),
			errors.WithProperty("solution_archive_name", solutionArchivePart.SolutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchivePart.SolutionArchive.Version),
		)
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(solutionArchivePart.SolutionArchive)
	fileSize, err := uc.archiveLister.GetArchiveSize(solutionArchiveFileName)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(201),
			errors.WithDetail("error on getting size of the solution archive"),
			errors.WithProperty("usecase", "download_solution_archive"),
			errors.WithProperty("solution_archive_name", solutionArchivePart.SolutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchivePart.SolutionArchive.Version),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchivePart.SolutionArchive.Name,
				solutionArchivePart.SolutionArchive.Version,
			)),
		)
	}

	start := solutionArchivePart.Meta.Start
	end := solutionArchivePart.Meta.End
	if start >= fileSize || end >= fileSize {
		return nil, errors.Wrap(domain.ErrHandlerInvalidRequestHeaderFormat,
			errors.WithIdentifier(202),
			errors.WithDetail("requested byte range exceeds the solution archive size"),
			errors.WithProperty("usecase", "download_solution_archive"),
			errors.WithProperty("range_start", start),
			errors.WithProperty("range_end", end),
			errors.WithProperty("file_size_bytes", fileSize),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchivePart.SolutionArchive.Name,
				solutionArchivePart.SolutionArchive.Version,
			)),
		)
	}

	// Retrieve the requested part of the solution archive from the storage
	file, err := uc.archiveReader.GetPart(
		solutionArchiveFileName,
		solutionArchivePart.Meta.Start,
		solutionArchivePart.Meta.End,
	)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(203),
			errors.WithDetail("error on getting part of the solution archive"),
			errors.WithProperty("usecase", "download_solution_archive"),
			errors.WithProperty("solution_archive_name", solutionArchivePart.SolutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchivePart.SolutionArchive.Version),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchivePart.SolutionArchive.Name,
				solutionArchivePart.SolutionArchive.Version,
			)),
		)
	}

	uc.logger.Debug().Any("solution_archive_part", solutionArchivePart).Msg("Solution archive chunk downloaded")

	return &domain.SolutionArchiveFile{
		File:          file,
		Size:          fileSize,
		ContentLength: solutionArchivePart.Meta.Size(),
	}, nil
}
