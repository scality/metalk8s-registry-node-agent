package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivesaver"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) GetFileSystemArchiveSaver() service.ArchiveSaver {
	if c.archiveSaver == nil {
		c.archiveSaver = archivesaver.NewFileSystem(
			c.GetLogger(),
			c.config.SolutionArchivesLocation,
		)
	}
	return c.archiveSaver
}

func (c *Container) GetMockFileSystemArchiveSaver() service.ArchiveSaver {
	if c.archiveSaver == nil {
		c.archiveSaver = c.GetMockFSSolutionArchiveStorage().(service.ArchiveSaver)
	}
	return c.archiveSaver
}
