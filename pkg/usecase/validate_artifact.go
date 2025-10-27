package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type ValidateArtifact struct {
	logger *zerolog.Logger

	artifactValidator service.ArtifactValidator
}

func NewValidateArtifact(
	logger *zerolog.Logger,
	artifactValidator service.ArtifactValidator,
) *ValidateArtifact {
	l := logger.With().Str("use_case", "validate_artifact").Logger()

	return &ValidateArtifact{
		logger:            &l,
		artifactValidator: artifactValidator,
	}
}

func (uc *ValidateArtifact) Execute(artifact *domain.Artifact) (bool, error) {
	uc.logger.Debug().
		Any("artifact", artifact).
		Msg("Validating artifact")

	isValid, err := uc.artifactValidator.ValidateArtifact(artifact)
	if err != nil {
		return false, errors.Intercept(err).
			WithDetail("failed to validate artifact").
			Throw()
	}

	uc.logger.Debug().Any("artifact", artifact).Msg("Artifact validated")

	return isValid, nil
}
