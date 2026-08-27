package di

import (
	"log/slog"
	"os"
	"regexp"

	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/storageprovider"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

const solutionArchiveStorageNamePattern = "^.{1,251}\\.iso$"

var solutionArchiveStorageNameRegexp = regexp.MustCompile(solutionArchiveStorageNamePattern)

func (c *Container) GetFSSolutionArchiveStorage() service.StorageProvider {
	if c.solutionArchiveStorage == nil {
		c.solutionArchiveStorage = storageprovider.NewFileSystem(
			&storageprovider.FileOpts{
				SolutionArchivesLocation:   c.config.SolutionArchivesLocation,
				SolutionsLocation:          c.config.SolutionsLocation,
				InterestContentFilterRegex: solutionArchiveStorageNameRegexp,
				Logger:                     c.GetLogger(),
			},
		)

		err := c.solutionArchiveStorage.Init()
		if err != nil {
			c.GetLogger().ErrorContext(c.ctx, "could not initialize solution archives storage", slog.Any("error", err))
			os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
		}

		c.GetLogger().InfoContext(c.ctx, "solution archives storage initialized")
	}

	return c.solutionArchiveStorage
}

func (c *Container) GetMockFSSolutionArchiveStorage() service.StorageProvider {
	if c.solutionArchiveStorage == nil {
		c.solutionArchiveStorage = storageprovider.NewMockFileSystem(
			&storageprovider.MockFileOpts{
				SolutionArchivesLocation:   c.config.SolutionArchivesLocation,
				SolutionsLocation:          c.config.SolutionsLocation,
				InterestContentFilterRegex: solutionArchiveStorageNameRegexp,
				Logger:                     c.GetLogger(),
			},
		)

		err := c.solutionArchiveStorage.Init()
		if err != nil {
			c.GetLogger().ErrorContext(c.ctx, "could not initialize solution archives storage", slog.Any("error", err))
			os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
		}

		c.GetLogger().InfoContext(c.ctx, "solution archives storage initialized")
	}

	return c.solutionArchiveStorage
}
