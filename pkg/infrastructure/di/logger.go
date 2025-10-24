package di

import (
	"github.com/rs/zerolog"

	"github.com/scality/metalk8s-registry-node-agent/cmd/config"
	logger "github.com/scality/platform-library/pkg/infrastructure/logger/zerolog"
)

func (c *Container) GetLogger() *zerolog.Logger {
	if c.logger == nil {
		l := logger.NewZerolog(&c.config.Logger).
			With().
			Str("application_name", config.ApplicationName).
			Str("application_version", config.ApplicationVersion).
			Logger()

		c.logger = &l
	}

	return c.logger
}
