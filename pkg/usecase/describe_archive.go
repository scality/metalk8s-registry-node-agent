// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type DescribeSolutionArchive struct {
	logger        *zerolog.Logger
	archiveLister service.ArchiveLister
	archiveLocker service.LockerUnlocker
	rootAPIPath   string
}

func NewDescribeSolutionArchive(
	logger *zerolog.Logger,
	archiveLister service.ArchiveLister,
	archiveLocker service.LockerUnlocker,
	rootAPIPath string,
) *DescribeSolutionArchive {
	l := logger.With().Str("use_case", "describe_solution_archive").Logger()

	return &DescribeSolutionArchive{
		logger:        &l,
		archiveLister: archiveLister,
		archiveLocker: archiveLocker,
		rootAPIPath:   rootAPIPath,
	}
}

func (uc *DescribeSolutionArchive) Execute(
	solutionArchive *domain.SolutionArchive,
) (int64, error) {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Describing solution archive")

	uc.archiveLocker.RLock(solutionArchive)
	defer uc.archiveLocker.RUnlock(solutionArchive)

	properties := solutionArchive.GetErrorProperties(
		"describe_solution_archive",
		solutionArchive.GetDownloadURL(uc.rootAPIPath),
	)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return 0, errors.Wrap(err,
			errors.WithIdentifier(196),
			errors.WithDetail("error while listing solution archives"),
			errors.WithProperties(properties),
		)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchive, fileNames) {
		return 0, errors.Wrap(domain.ErrNotFound,
			errors.WithIdentifier(197),
			errors.WithDetail("solution archive not found"),
			errors.WithProperties(properties),
		)
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(solutionArchive)
	size, err := uc.archiveLister.GetArchiveSize(solutionArchiveFileName)
	if err != nil {
		return 0, errors.Wrap(err,
			errors.WithIdentifier(198),
			errors.WithDetail("error while getting archive size"),
			errors.WithProperties(properties),
		)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive described")

	return size, nil
}
