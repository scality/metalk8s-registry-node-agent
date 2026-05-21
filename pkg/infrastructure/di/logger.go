package di

import (
	"log"
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

		hostname, err := os.Hostname()
		if err != nil {
			log.Fatalf("failed to get hostname: %v", err)
		}

		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})

		c.logger = slog.New(handler).With(
			slog.String("application_name", config.ApplicationName),
			slog.String("application_version", config.ApplicationVersion),
			slog.String("host", hostname),
		)
	}

	return c.logger
}
