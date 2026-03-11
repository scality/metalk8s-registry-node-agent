package archivedownloader

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type Storage struct {
	store       service.StorageProvider
	logger      *zerolog.Logger
	rootAPIPath string
}

func NewStorage(
	store service.StorageProvider,
	logger *zerolog.Logger,
	rootAPIPath string,
) *Storage {
	l := logger.With().Str("infrastructure", "archive_downloader").Logger()
	return &Storage{
		store:       store,
		logger:      &l,
		rootAPIPath: rootAPIPath,
	}
}

var _ service.SolutionArchiveDownloader = &Storage{}

func (s *Storage) DownloadSolutionArchive(
	solutionArchivePart *domain.Part,
) (*domain.SolutionArchiveFile, error) {
	s.store.Lock()
	defer s.store.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := s.store.ListFiles()
	if err != nil {
		return nil, errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchivePart.SolutionArchive, fileNames) {
		return nil, errors.From(domain.ErrNotFound).
			WithDetail("solution archive not found").
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", s.rootAPIPath, solutionArchivePart.SolutionArchive.Name)).
			WithProperty("solution_archive_name", solutionArchivePart.SolutionArchive.Name).
			WithProperty("solution_archive_version", solutionArchivePart.SolutionArchive.Version).
			Throw()
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(solutionArchivePart.SolutionArchive)
	fileSize, err := s.store.GetSizeFromFileInfos(solutionArchiveFileName)
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
	file, err := s.store.GetPart(solutionArchiveFileName, solutionArchivePart.Meta.Start, solutionArchivePart.Meta.End)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return &domain.SolutionArchiveFile{
		File: file,
		Size: solutionArchivePart.Meta.Size(),
	}, nil
}
