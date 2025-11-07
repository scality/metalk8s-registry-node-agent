package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/solutioncleaner"

func (c *Container) getStorageSolutionCleaner() *solutioncleaner.Storage {
	if c.storageSolutionCleaner == nil {
		c.storageSolutionCleaner = solutioncleaner.NewStorage(
			c.GetFSSolutionArchiveStorage(),
			c.GetLogger(),
			c.config.SolutionsLocation,
		)
	}

	return c.storageSolutionCleaner
}
