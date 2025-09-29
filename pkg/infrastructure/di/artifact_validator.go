package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/artifactvalidator"

func (c *Container) getStorageArtifactValidator() *artifactvalidator.Storage {
	if c.storageArtifactValidator == nil {
		c.storageArtifactValidator = artifactvalidator.NewStorage(
			c.GetFileSystemArtifactStorage(),
			c.GetLogger(),
		)
	}

	return c.storageArtifactValidator
}
