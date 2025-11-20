package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type UploadPart struct {
	logger       *zerolog.Logger
	partUploader service.PartUploader
}

func NewUploadPart(
	logger *zerolog.Logger,
	partUploader service.PartUploader,
) *UploadPart {
	l := logger.With().Str("use_case", "upload_part").Logger()

	return &UploadPart{
		logger:       &l,
		partUploader: partUploader,
	}
}

func (uc *UploadPart) Execute(part *domain.Part) (*domain.SolutionArchiveStatus, error) {
	uc.logger.Info().Msg("Uploading part")

	partStatus, err := uc.partUploader.UploadPart(part)
	if err != nil {
		return nil, errors.Intercept(err).
			WithDetail("failed to upload part").
			Throw()
	}

	uc.logger.Info().Msg("Part uploaded")

	return partStatus, nil
}
