// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type DownloadSolutionArchive struct {
	logger *zerolog.Logger
	locker sync.Locker

	fileLister  service.FileLister
	fileReader  service.FileReader
	rootAPIPath string
}

func NewDownloadSolutionArchive(
	logger *zerolog.Logger,
	locker sync.Locker,
	fileLister service.FileLister,
	fileReader service.FileReader,
	rootAPIPath string,
) *DownloadSolutionArchive {
	l := logger.With().Str("use_case", "download_solution_archive").Logger()

	return &DownloadSolutionArchive{
		logger:      &l,
		locker:      locker,
		fileLister:  fileLister,
		fileReader:  fileReader,
		rootAPIPath: rootAPIPath,
	}
}

func (uc *DownloadSolutionArchive) Execute(
	solutionArchivePart *domain.Part,
) (*domain.SolutionArchiveFile, error) {
	uc.logger.Debug().
		Any("solution_archive_part", solutionArchivePart).
		Msg("Downloading solution archive chunk")

	uc.locker.Lock()
	defer uc.locker.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.fileLister.ListFiles()
	if err != nil {
		return nil, errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchivePart.SolutionArchive, fileNames) {
		return nil, errors.From(domain.ErrNotFound).
			WithDetail("solution archive not found").
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, solutionArchivePart.SolutionArchive.Name)).
			WithProperty("solution_archive_name", solutionArchivePart.SolutionArchive.Name).
			WithProperty("solution_archive_version", solutionArchivePart.SolutionArchive.Version).
			Throw()
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(solutionArchivePart.SolutionArchive)
	fileSize, err := uc.fileLister.GetSizeFromFileInfos(solutionArchiveFileName)
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
	file, err := uc.fileReader.GetPart(solutionArchiveFileName, solutionArchivePart.Meta.Start, solutionArchivePart.Meta.End)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	uc.logger.Debug().Any("solution_archive_part", solutionArchivePart).Msg("Solution archive chunk downloaded")

	return &domain.SolutionArchiveFile{
		File: file,
		Size: solutionArchivePart.Meta.Size(),
	}, nil
}
