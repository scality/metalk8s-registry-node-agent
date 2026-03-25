package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type MountSolutionArchive struct {
	logger      *zerolog.Logger
	fileLister  service.FileLister
	fileMounter service.FileMounter
}

func NewMountSolutionArchive(
	logger *zerolog.Logger,
	fileLister service.FileLister,
	fileMounter service.FileMounter,
) *MountSolutionArchive {
	l := logger.With().Str("use_case", "mount_solution_archive").Logger()

	return &MountSolutionArchive{
		logger:      &l,
		fileLister:  fileLister,
		fileMounter: fileMounter,
	}
}

func (uc *MountSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Mounting solution archive")

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.fileLister.ListFiles()
	if err != nil {
		return errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		err := uc.fileMounter.MountFile(library.GenSolutionArchiveFileName(solutionArchive),
			library.GenSolutionDirName(solutionArchive))
		if err != nil {
			return errors.Stamp(err)
		}

		return nil
	}

	if !library.SolutionArchiveExists(solutionArchive, fileNames) {
		return errors.From(domain.ErrNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found").
			WithProperty("solution_archive_name", solutionArchive.Name).
			WithProperty("solution_archive_version", solutionArchive.Version).
			Throw()
	}

	err = uc.fileMounter.MountFile(library.GenSolutionArchiveFileName(solutionArchive),
		library.GenSolutionDirName(solutionArchive))
	if err != nil {
		return errors.Stamp(err)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive mounted")

	return nil
}
