package di

import (
	archivecleaner "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivecleaner"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) getFileSystemArchiveCleaner() service.ArchiveCleaner {
	if c.archiveCleaner == nil {
		c.archiveCleaner = archivecleaner.NewFileSystem(
			c.GetLogger(),
			c.GetFSSolutionArchiveStorage(),
			c.getFileSystemArchiveMounter(),
		)
	}
	return c.archiveCleaner
}

func (c *Container) GetMockFileSystemArchiveCleaner() service.ArchiveCleaner {
	if c.archiveCleaner == nil {
		c.archiveCleaner = c.GetMockFSSolutionArchiveStorage().(service.ArchiveCleaner)
	}
	return c.archiveCleaner
}
