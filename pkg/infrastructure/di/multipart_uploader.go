package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/multipartuploader"

func (c *Container) GetFilesystemMultipartUploader() *multipartuploader.FileSystem {
	if c.filesystemMultipartUploader == nil {
		lifecycle := c.GetStorageLifecycle()
		c.filesystemMultipartUploader = multipartuploader.NewFileSystem(
			c.GetLogger(),
			lifecycle.SolutionArchivesLocation(),
		)
	}

	return c.filesystemMultipartUploader
}
