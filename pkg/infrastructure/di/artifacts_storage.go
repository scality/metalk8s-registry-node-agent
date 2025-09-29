package di

import (
	"regexp"

	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/storageprovider"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

const artifactStorageNamePattern = "^.{1,251}\\.iso$"

var artifactStorageNameRegexp = regexp.MustCompile(artifactStorageNamePattern)

func (c *Container) GetFileSystemArtifactStorage() service.StorageProvider {
	if c.artifactStorage == nil {
		c.artifactStorage = storageprovider.NewFileSystem(
			&storageprovider.FileOpts{
				RootLocation:               c.config.ArtifactStorageRootLocation,
				InterestContentFilterRegex: artifactStorageNameRegexp,
				Logger:                     c.GetLogger(),
			},
		)

		err := c.artifactStorage.Init()
		if err != nil {
			c.GetLogger().Fatal().Err(err).Msg("could not initialize artifacts storage")
		}

		c.GetLogger().Info().Msg("Artifact storage initialized")
	}

	return c.artifactStorage
}
