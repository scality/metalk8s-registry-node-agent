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

type DescribeSolutionArchive struct {
	logger *zerolog.Logger

	locker      sync.Locker
	fileLister  service.FileLister
	rootAPIPath string
}

func NewDescribeSolutionArchive(
	logger *zerolog.Logger,
	locker sync.Locker,
	fileLister service.FileLister,
	rootAPIPath string,
) *DescribeSolutionArchive {
	l := logger.With().Str("use_case", "describe_solution_archive").Logger()

	return &DescribeSolutionArchive{
		logger:      &l,
		locker:      locker,
		fileLister:  fileLister,
		rootAPIPath: rootAPIPath,
	}
}

func (uc *DescribeSolutionArchive) Execute(
	solutionArchive *domain.SolutionArchive,
) (int64, error) {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Describing solution archive")

	uc.locker.Lock()
	defer uc.locker.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.fileLister.ListFiles()
	if err != nil {
		return 0, errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if !library.SolutionArchiveExists(solutionArchive, fileNames) {
		return 0, errors.From(domain.ErrNotFound).
			WithDetail("solution archive not found").
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, solutionArchive.Name)).
			WithProperty("solution_archive_name", solutionArchive.Name).
			WithProperty("solution_archive_version", solutionArchive.Version).
			Throw()
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(solutionArchive)
	size, err := uc.fileLister.GetSizeFromFileInfos(solutionArchiveFileName)
	if err != nil {
		return 0, errors.Stamp(err)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive described")

	return size, nil
}
