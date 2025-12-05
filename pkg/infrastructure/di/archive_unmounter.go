package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archiveunmounter"

func (c *Container) getStorageSolutionArchiveUnmounter() *archiveunmounter.Storage {
	if c.storageSolutionArchiveUnmounter == nil {
		c.storageSolutionArchiveUnmounter = archiveunmounter.NewStorage(
			c.GetFSSolutionArchiveStorage(),
			c.GetLogger(),
		)
	}

	return c.storageSolutionArchiveUnmounter
}
