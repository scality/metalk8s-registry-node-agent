package di

import (
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
				SolutionArchiveLocation:    c.config.SolutionArchivesLocation,
				InterestContentFilterRegex: solutionArchiveStorageNameRegexp,
				Logger:                     c.GetLogger(),
			},
		)

		err := c.solutionArchiveStorage.Init()
		if err != nil {
			c.GetLogger().Fatal().Err(err).Msg("could not initialize solution archives storage")
		}

		c.GetLogger().Info().Msg("solution archives storage initialized")
	}

	return c.solutionArchiveStorage
}

func (c *Container) GetMockFSSolutionArchiveStorage() service.StorageProvider {
	if c.solutionArchiveStorage == nil {
		c.solutionArchiveStorage = storageprovider.NewMockFileSystem(
			&storageprovider.MockFileOpts{
				SolutionArchiveLocation:    c.config.SolutionArchivesLocation,
				InterestContentFilterRegex: solutionArchiveStorageNameRegexp,
				Logger:                     c.GetLogger(),
			},
		)

		err := c.solutionArchiveStorage.Init()
		if err != nil {
			c.GetLogger().Fatal().Err(err).Msg("could not initialize solution archives storage")
		}

		c.GetLogger().Info().Msg("solution archives storage initialized")
	}

	return c.solutionArchiveStorage
}
