package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/lockerunlocker"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) getInMemoryArchiveLocker() service.LockerUnlocker {
	if c.inMemoryArchiveLocker == nil {
		c.inMemoryArchiveLocker = lockerunlocker.NewInMemory()
	}

	return c.inMemoryArchiveLocker
}
