package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/bucketmanager"

func (c *Container) GetFilesystemBucketManager() *bucketmanager.FileSystem {
	if c.filesystemBucketManager == nil {
		lifecycle := c.GetStorageLifecycle()
		c.filesystemBucketManager = bucketmanager.NewFileSystem(
			c.GetLogger(),
			lifecycle.SolutionArchivesLocation(),
		)
	}

	return c.filesystemBucketManager
}
