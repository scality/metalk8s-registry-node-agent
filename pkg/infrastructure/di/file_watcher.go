package di

import (
	"log/slog"
	"os"

	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/filewatcher"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) GetFileSystemFileWatcher() service.FileWatcher {
	if c.fileWatcher == nil {
		c.fileWatcher = filewatcher.NewFileSystem(
			c.GetLogger(),
			c.GetFSSolutionArchiveStorage(),
			c.config.SolutionArchivesLocation,
			c.config.SolutionsLocation,
		)
		err := c.fileWatcher.Init()
		if err != nil {
			c.GetLogger().ErrorContext(c.ctx, "could not initialize file watcher", slog.Any("error_message", err))
			os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
		}
	}

	return c.fileWatcher
}

func (c *Container) GetMockFileSystemFileWatcher() service.FileWatcher {
	if c.fileWatcher == nil {
		c.fileWatcher = c.GetMockFSSolutionArchiveStorage().(service.FileWatcher)
		err := c.fileWatcher.Init()
		if err != nil {
			c.GetLogger().ErrorContext(c.ctx, "could not initialize file watcher", slog.Any("error_message", err))
			os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
		}
	}
	return c.fileWatcher
}
