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

		ChunkSizeMB int64 `env:"CHUNK_SIZE_MB, default=10"`
	}

	Extern struct {
		Addr        string      `env:"ADDR, default=:5001"`
		ServerTLS   ServerTLS   `env:",prefix=SERVER_TLS_"`
		ServerAuthN ServerAuthN `env:",prefix=SERVER_AUTHN_"`
	}

	Intern struct {
		Addr        string      `env:"ADDR, default=:5002"`
		ServerTLS   ServerTLS   `env:",prefix=SERVER_TLS_"`
		ServerAuthN ServerAuthN `env:",prefix=SERVER_AUTHN_"`
		ClientTLS   ClientTLS   `env:",prefix=CLIENT_TLS_"`
		ClientAuthN ClientAuthN `env:",prefix=CLIENT_AUTHN_"`
	}

	ServerTLS struct {
		// identical for Intern and Extern
		CertFilePath string `env:"CERT_FILE_PATH"`
		KeyFilePath  string `env:"KEY_FILE_PATH"`
	}
	ServerAuthN struct {
		// identical for Intern and Extern
		CACertFilePath string `env:"CA_CERT_FILE_PATH"`
	}

	ClientTLS struct {
		CACertFilePath string `env:"CA_CERT_FILE_PATH"`
	}

	ClientAuthN struct {
		CertFilePath string `env:"CERT_FILE_PATH"`
		KeyFilePath  string `env:"KEY_FILE_PATH"`
	}
)

func NewEnvironment(ctx context.Context) (*Environment, error) {
	cfg := &Environment{
		RootExternAPIPath: rootExternAPIPath,
		RootInternAPIPath: rootInternAPIPath,
	}

	err := cfg.Load(ctx)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(3),
		)
	}

	return cfg, nil
}

func (cfg *Environment) Load(ctx context.Context) error {
	err := envconfig.Process(ctx, cfg)
	if err != nil {
		return errors.Wrap(domain.ErrConfigurationLoading,
			errors.WithIdentifier(1),
			errors.WithDetail("failed to process environment variables"),
			errors.CausedBy(err),
		)
	}

	if cfg.ChunkSizeMB <= 0 {
		return errors.Wrap(domain.ErrConfigurationLoading,
			errors.WithIdentifier(2),
			errors.WithDetail("chunk size must be greater than 0"),
		)
	}

	return nil
}
