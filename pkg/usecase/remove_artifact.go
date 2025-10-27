package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type RemoveArtifact struct {
	logger *zerolog.Logger

	artifactRemover service.ArtifactRemover
}

func NewRemoveArtifact(
	logger *zerolog.Logger,
	artifactRemover service.ArtifactRemover,
) *RemoveArtifact {
	l := logger.With().Str("use_case", "remove_artifact").Logger()

	return &RemoveArtifact{
		logger:          &l,
		artifactRemover: artifactRemover,
	}
}

func (uc *RemoveArtifact) Execute(artifact *domain.Artifact) error {
	uc.logger.Debug().
		Any("artifact", artifact).
		Msg("Removing artifact")

	err := uc.artifactRemover.RemoveArtifact(artifact)
	if err != nil {
		return errors.Intercept(err).
			WithDetail("failed to remove artifact").
			Throw()
	}

	uc.logger.Debug().Any("artifact", artifact).Msg("Artifact removed")

	return nil
}
