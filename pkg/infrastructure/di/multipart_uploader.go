package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/multipartuploader"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) GetFileSystemMultipartUploader() service.MultipartUploader {
	if c.multipartUploader == nil {
		c.multipartUploader = multipartuploader.NewFileSystem(
			c.GetLogger(),
			c.config.SolutionArchivesLocation,
			c.GetFileSystemMultipartInspector(),
		)
	}
	return c.multipartUploader
}

func (c *Container) GetMockFileSystemMultipartUploader() service.MultipartUploader {
	if c.multipartUploader == nil {
		c.multipartUploader = c.GetMockFSSolutionArchiveStorage().(service.MultipartUploader)
	}
	return c.multipartUploader
}
