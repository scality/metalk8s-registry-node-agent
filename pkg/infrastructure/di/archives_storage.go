package di

import (
	"regexp"

	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/storageprovider"
)

const solutionArchiveStorageNamePattern = "^.{1,251}\\.iso$"

var solutionArchiveStorageNameRegexp = regexp.MustCompile(solutionArchiveStorageNamePattern)

func (c *Container) GetStorageLifecycle() *storageprovider.FileSystem {
	if c.storageLifecycle == nil {
		c.storageLifecycle = storageprovider.NewFileSystem(
			&storageprovider.FileOpts{
				SolutionArchivesLocation:   c.config.SolutionArchivesLocation,
				SolutionsLocation:          c.config.SolutionsLocation,
				InterestContentFilterRegex: solutionArchiveStorageNameRegexp,
				Logger:                     c.GetLogger(),
			},
		)

		err := c.storageLifecycle.Init()
		if err != nil {
			c.GetLogger().Fatal().Err(err).Msg("could not initialize solution archives storage")
		}

		c.GetLogger().Info().Msg("solution archives storage initialized")
	}

	return c.storageLifecycle
}
