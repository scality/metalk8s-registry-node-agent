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

const (
	ApplicationName   = "metalk8s-registry-node-agent"
	rootExternAPIPath = "/api/v1"
	rootInternAPIPath = "/api/v1"
)

type (
	Environment struct {
		Logger logger.Config `env:",prefix=LOGGER_"`

		SolutionArchivesLocation string `env:"SOLUTION_ARCHIVES_LOCATION, default=/archives"`
		SolutionsLocation        string `env:"SOLUTIONS_LOCATION, default=/solutions"`

		Extern Extern `env:",prefix=EXTERN_"`
		Intern Intern `env:",prefix=INTERN_"`

		RootExternAPIPath string
		RootInternAPIPath string
	}

	Extern struct {
		Addr string `env:"ADDR, default=:5001"`
	}

	Intern struct {
		Addr string `env:"ADDR, default=:5002"`
	}
)

func NewEnvironment(ctx context.Context) (*Environment, error) {
	cfg := &Environment{
		RootExternAPIPath: rootExternAPIPath,
		RootInternAPIPath: rootInternAPIPath,
	}

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
			WithDetail("failed to process environment variables").
			CausedBy(err).
			Throw()
	}

	return nil
}
