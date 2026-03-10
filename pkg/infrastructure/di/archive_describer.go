package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivedescriber"

func (c *Container) getStorageSolutionArchiveDescriber() *archivedescriber.Storage {
	if c.storageSolutionArchiveDescriber == nil {
		c.storageSolutionArchiveDescriber = archivedescriber.NewStorage(
			c.GetFSSolutionArchiveStorage(),
			c.GetLogger(),
			c.GetRootInternAPIPath(),
		)
	}

	return c.storageSolutionArchiveDescriber
}
