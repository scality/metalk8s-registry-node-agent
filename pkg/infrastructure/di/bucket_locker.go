package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/bucketlocker"

func (c *Container) getInMemoryBucketLocker() *bucketlocker.InMemory {
	if c.inMemoryBucketLocker == nil {
		c.inMemoryBucketLocker = bucketlocker.NewInMemory()
	}

	return c.inMemoryBucketLocker
}
