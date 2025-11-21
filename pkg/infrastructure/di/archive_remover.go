package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archiveremover"
)

func (c *Container) getStorageSolutionArchiveRemover() *archiveremover.Storage {
	if c.storageSolutionArchiveRemover == nil {
		c.storageSolutionArchiveRemover = archiveremover.NewStorage(
			c.GetFSSolutionArchiveStorage(),
			c.GetLogger(),
		)
	}

	return c.storageSolutionArchiveRemover
}
