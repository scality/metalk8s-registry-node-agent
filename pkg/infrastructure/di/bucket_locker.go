package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/lockerunlocker"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) getInMemoryBucketLocker() service.LockerUnlocker {
	if c.inMemoryBucketLocker == nil {
		c.inMemoryBucketLocker = lockerunlocker.NewInMemory()
	}

	return c.inMemoryBucketLocker
}
