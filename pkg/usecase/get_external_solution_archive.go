package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type GetExternalSolutionArchive struct {
	logger             *zerolog.Logger
	externalDownloader service.ExternalDownloader
	archiveLister      service.ArchiveLister
	archiveSaver       service.ArchiveSaver
	archiveLocker      service.LockerUnlocker
}

func NewGetExternalSolutionArchive(
	logger *zerolog.Logger,
	externalDownloader service.ExternalDownloader,
	archiveLister service.ArchiveLister,
	archiveSaver service.ArchiveSaver,
	archiveLocker service.LockerUnlocker,
) *GetExternalSolutionArchive {
	l := logger.With().Str("use_case", "get_external_solution_archive").Logger()

	return &GetExternalSolutionArchive{
		logger:             &l,
		externalDownloader: externalDownloader,
		archiveLister:      archiveLister,
		archiveSaver:       archiveSaver,
		archiveLocker:      archiveLocker,
	}
}

func (uc *GetExternalSolutionArchive) Execute(solutionArchive *domain.SolutionArchive, downloadURL string) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Str("download_url", downloadURL).
		Msg("Getting external solution archive")

	// To avoid simultaneous downloads and deletions of the same solution archive
	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		return nil
	}

	body, err := uc.externalDownloader.Download(downloadURL)
	if err != nil {
		return errors.Stamp(err)
	}

	err = uc.archiveSaver.SaveFile(
		library.GenSolutionArchiveFileName(solutionArchive),
		body,
		library.FileSystemDefaultFileMode,
	)
	if err != nil {
		return errors.Stamp(err)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("External solution archive retrieved")

	return nil
}
