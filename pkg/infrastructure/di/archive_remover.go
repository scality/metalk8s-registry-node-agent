package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archiveremover"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) getFileSystemArchiveRemover() service.ArchiveRemover {
	if c.archiveRemover == nil {
		c.archiveRemover = archiveremover.NewFileSystem(
			c.GetLogger(),
			c.config.SolutionArchivesLocation,
		)
	}
	return c.archiveRemover
}

func (c *Container) GetMockFileSystemArchiveRemover() service.ArchiveRemover {
	if c.archiveRemover == nil {
		c.archiveRemover = c.GetMockFSSolutionArchiveStorage().(service.ArchiveRemover)
	}
	return c.archiveRemover
}
