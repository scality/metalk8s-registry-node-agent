package di

import (
	"log/slog"
	"os"

	"github.com/scality/metalk8s-registry-node-agent/cmd/config"
)

func (c *Container) GetLogger() *slog.Logger {
	if c.logger == nil {
		var level slog.Level
		if err := level.UnmarshalText([]byte(c.config.Logger.LogLevel)); err != nil {
			level = slog.LevelInfo
		}

		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})

		logger := slog.New(handler).With(
			slog.String("application_name", config.ApplicationName),
			slog.String("application_version", config.ApplicationVersion),
		)

		if hostname, err := os.Hostname(); err == nil {
			logger = logger.With(slog.String("host", hostname))
		}

		c.logger = logger
	}

	return c.logger
}
