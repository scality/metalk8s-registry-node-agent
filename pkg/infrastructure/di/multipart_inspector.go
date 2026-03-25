package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/multipartinspector"

func (c *Container) GetFilesystemMultipartInspector() *multipartinspector.FileSystem {
	if c.filesystemMultipartInspector == nil {
		lifecycle := c.GetFilesystemStorageLifecycle()
		c.filesystemMultipartInspector = multipartinspector.NewFileSystem(
			c.GetLogger(),
			lifecycle.SolutionArchivesLocation(),
			c.getInMemoryBucketLocker(),
		)
	}

	return c.filesystemMultipartInspector
}
