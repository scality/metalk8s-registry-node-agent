// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"context"
	"log/slog"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type DescribeSolutionArchive struct {
	logger        *slog.Logger
	archiveLister service.ArchiveLister
	archiveLocker service.LockerUnlocker
	rootAPIPath   string
}

func NewDescribeSolutionArchive(
	logger *slog.Logger,
	archiveLister service.ArchiveLister,
	archiveLocker service.LockerUnlocker,
	rootAPIPath string,
) *DescribeSolutionArchive {
	return &DescribeSolutionArchive{
		logger:        logger.With(slog.String("use_case", "describe_solution_archive")),
		archiveLister: archiveLister,
		archiveLocker: archiveLocker,
		rootAPIPath:   rootAPIPath,
	}
}

func (uc *DescribeSolutionArchive) Execute(
	ctx context.Context,
	solutionArchive *domain.SolutionArchive,
) (int64, error) {
	uc.logger.DebugContext(ctx, "Describing solution archive",
		slog.Any("solution_archive", solutionArchive),
	)

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

	uc.logger.DebugContext(ctx, "Solution archive described",
		slog.Any("solution_archive", solutionArchive),
	)

	return size, nil
}
