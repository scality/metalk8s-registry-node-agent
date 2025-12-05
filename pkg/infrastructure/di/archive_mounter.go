package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivemounter"

func (c *Container) getStorageSolutionArchiveMounter() *archivemounter.Storage {
	if c.storageSolutionArchiveMounter == nil {
		c.storageSolutionArchiveMounter = archivemounter.NewStorage(
			c.GetFSSolutionArchiveStorage(),
			c.GetLogger(),
		)
	}

	return c.storageSolutionArchiveMounter
}
