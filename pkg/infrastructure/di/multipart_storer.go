package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/multipartstorer"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) GetFileSystemMultipartStorer() service.MultipartStorer {
	if c.multipartStorer == nil {
		c.multipartStorer = multipartstorer.NewFileSystem(
			c.GetLogger(),
			c.config.SolutionArchivesLocation,
			c.GetFileSystemMultipartInspector(),
			c.GetFileSystemMultipartRemover(),
			c.GetFileSystemBucketManager(),
		)
	}
	return c.multipartStorer
}

func (c *Container) GetMockFileSystemMultipartStorer() service.MultipartStorer {
	if c.multipartStorer == nil {
		c.multipartStorer = c.GetMockFSSolutionArchiveStorage().(service.MultipartStorer)
	}
	return c.multipartStorer
}
