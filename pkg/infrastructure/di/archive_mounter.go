package di

import (
	archivemounter "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivemounter"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) getFileSystemArchiveMounter() service.ArchiveMounter {
	if c.archiveMounter == nil {
		c.archiveMounter = archivemounter.NewFileSystem(
			c.GetLogger(),
			c.config.SolutionArchivesLocation,
			c.config.SolutionsLocation,
			c.getFileSystemArchiveRemover(),
			c.GetFileSystemFileWatcher(),
			c.GetFileSystemMountWatcher(),
		)
	}
	return c.archiveMounter
}

func (c *Container) GetMockFileSystemArchiveMounter() service.ArchiveMounter {
	if c.archiveMounter == nil {
		c.archiveMounter = c.GetMockFSSolutionArchiveStorage().(service.ArchiveMounter)
	}
	return c.archiveMounter
}
