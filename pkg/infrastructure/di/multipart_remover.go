package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/multipartremover"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) getFileSystemMultipartRemover() service.MultipartRemover {
	if c.multipartRemover == nil {
		c.multipartRemover = multipartremover.NewFileSystem(
			c.GetLogger(),
			c.config.SolutionArchivesLocation,
		)
	}
	return c.multipartRemover
}

func (c *Container) GetMockFileSystemMultipartRemover() service.MultipartRemover {
	if c.multipartRemover == nil {
		c.multipartRemover = c.GetMockFSSolutionArchiveStorage().(service.MultipartRemover)
	}
	return c.multipartRemover
}
