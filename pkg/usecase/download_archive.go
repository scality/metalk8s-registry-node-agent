// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type DownloadSolutionArchive struct {
	logger *zerolog.Logger

	solutionArchiveGetter service.SolutionArchiveDownloader
}

func NewDownloadSolutionArchive(
	logger *zerolog.Logger,
	solutionArchiveGetter service.SolutionArchiveDownloader,
) *DownloadSolutionArchive {
	l := logger.With().Str("use_case", "download_solution_archive").Logger()

	return &DownloadSolutionArchive{
		logger:                &l,
		solutionArchiveGetter: solutionArchiveGetter,
	}
}

func (uc *DownloadSolutionArchive) Execute(
	solutionArchive *domain.SolutionArchive,
) (*domain.SolutionArchiveFile, error) {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Downloading solution archive")

	solutionArchiveFile, err := uc.solutionArchiveGetter.DownloadSolutionArchive(solutionArchive)
	if err != nil {
		return nil, errors.Intercept(err).
			WithDetail("failed to download solution archive").
			Throw()
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive downloaded")

	return solutionArchiveFile, nil
}
