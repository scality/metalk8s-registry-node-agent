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
	solutionArchivePart *domain.Part,
) (*domain.SolutionArchiveFile, error) {
	uc.logger.Debug().
		Any("solution_archive_part", solutionArchivePart).
		Msg("Downloading solution archive chunk")

	solutionArchiveFile, err := uc.solutionArchiveGetter.DownloadSolutionArchive(solutionArchivePart)
	if err != nil {
		return nil, errors.Intercept(err).
			WithDetail("failed to download solution archive chunk").
			Throw()
	}

	uc.logger.Debug().Any("solution_archive_part", solutionArchivePart).Msg("Solution archive chunk downloaded")

	return solutionArchiveFile, nil
}
