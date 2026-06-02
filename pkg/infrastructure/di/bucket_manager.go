package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/bucketmanager"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) getFileSystemBucketManager() service.BucketManager {
	if c.bucketManager == nil {
		c.bucketManager = bucketmanager.NewFileSystem(
			c.GetLogger(),
			c.config.SolutionArchivesLocation,
		)
	}
	return c.bucketManager
}

func (c *Container) GetMockFileSystemBucketManager() service.BucketManager {
	if c.bucketManager == nil {
		c.bucketManager = c.GetMockFSSolutionArchiveStorage().(service.BucketManager)
	}
	return c.bucketManager
}
