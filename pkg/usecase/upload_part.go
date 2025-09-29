package usecase

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

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

func (uc *UploadPart) Execute(part *domain.Part) (*domain.ArtifactStatus, error) {
	uc.logger.Info().Msg("Uploading part")

	partStatus, err := uc.partUploader.UploadPart(part)
	if err != nil {
		return nil, errors.Wrap(err, "failed to upload part")
	}

	uc.logger.Info().Msg("Part uploaded")

	return partStatus, nil
}
