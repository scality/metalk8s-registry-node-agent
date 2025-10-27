package config

import (
	"context"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	logger "github.com/scality/platform-library/pkg/infrastructure/logger/zerolog"
	"github.com/sethvargo/go-envconfig"
)

// ApplicationVersion is the version of the application.
// It is set at build time using ldflags.
//
//nolint:gochecknoglobals // This is a constant.
var ApplicationVersion = "dev"

const ApplicationName = "metalk8s-registry-node-agent"
const RootAPIPath = "/api/v1"

type (
	Environment struct {
		Logger logger.Config `env:",prefix=LOGGER_"`
		HTTP   HTTP          `env:",prefix=HTTP_"`

		ArtifactStorageRootLocation string `env:"ARTIFACT_STORAGE_ROOT_LOCATION, default=/artifacts"`
	}

	HTTP struct {
		Addr string `env:"ADDR, default=:5001"`
	}
)

func NewEnvironment(ctx context.Context) (*Environment, error) {
	cfg := &Environment{}

	err := cfg.Load(ctx)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return cfg, nil
}

func (cfg *Environment) Load(ctx context.Context) error {
	err := envconfig.Process(ctx, cfg)
	if err != nil {
		return errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("failed to process environment variables.").
			CausedBy(err).
			Throw()
	}

	return nil
}
