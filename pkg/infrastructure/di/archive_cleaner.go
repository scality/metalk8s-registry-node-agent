package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivecleaner"
)

func (c *Container) getStorageSolutionArchiveCleaner() *archivecleaner.Storage {
	if c.storageSolutionArchiveCleaner == nil {
		c.storageSolutionArchiveCleaner = archivecleaner.NewStorage(
			c.GetFSSolutionArchiveStorage(),
			c.GetLogger(),
		)
	}

	return c.storageSolutionArchiveCleaner
}
