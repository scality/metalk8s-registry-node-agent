package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/sessioninitializer"
)

func (c *Container) getStorageSessionInitializer() *sessioninitializer.Storage {
	if c.storageSessionInitializer == nil {
		c.storageSessionInitializer = sessioninitializer.NewStorage(
			c.GetFileSystemArtifactStorage(),
			c.GetLogger(),
		)
	}

	return c.storageSessionInitializer
}
