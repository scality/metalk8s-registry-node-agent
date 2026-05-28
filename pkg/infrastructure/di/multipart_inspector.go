package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/multipartinspector"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) getFileSystemMultipartInspector() service.MultipartInspector {
	if c.multipartInspector == nil {
		c.multipartInspector = multipartinspector.NewFileSystem(
			c.GetLogger(),
			c.config.SolutionArchivesLocation,
		)
	}
	return c.multipartInspector
}

func (c *Container) GetMockFileSystemMultipartInspector() service.MultipartInspector {
	if c.multipartInspector == nil {
		c.multipartInspector = c.GetMockFSSolutionArchiveStorage().(service.MultipartInspector)
	}
	return c.multipartInspector
}
