package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/artifactremover"

func (c *Container) getStorageArtifactRemover() *artifactremover.Storage {
	if c.storageArtifactRemover == nil {
		c.storageArtifactRemover = artifactremover.NewStorage(
			c.GetFileSystemArtifactStorage(),
			c.GetLogger(),
		)
	}

	return c.storageArtifactRemover
}
