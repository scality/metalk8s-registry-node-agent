package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/sessionremover"

func (c *Container) getStorageSessionRemover() *sessionremover.Storage {
	if c.storageSessionRemover == nil {
		c.storageSessionRemover = sessionremover.NewStorage(
			c.GetFSSolutionArchiveStorage(),
			c.GetLogger(),
		)
	}

	return c.storageSessionRemover
}
