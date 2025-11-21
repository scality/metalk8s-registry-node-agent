package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivevalidator"

func (c *Container) getStorageSolutionArchiveValidator() *archivevalidator.Storage {
	if c.storageSolutionArchiveValidator == nil {
		c.storageSolutionArchiveValidator = archivevalidator.NewStorage(
			c.GetFSSolutionArchiveStorage(),
			c.GetLogger(),
		)
	}

	return c.storageSolutionArchiveValidator
}
