package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivereader"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) getFileSystemArchiveReader() service.ArchiveReader {
	if c.archiveReader == nil {
		c.archiveReader = archivereader.NewFileSystem(
			c.GetLogger(),
			c.config.SolutionArchivesLocation,
		)
	}
	return c.archiveReader
}

func (c *Container) GetMockFileSystemArchiveReader() service.ArchiveReader {
	if c.archiveReader == nil {
		c.archiveReader = c.GetMockFSSolutionArchiveStorage().(service.ArchiveReader)
	}
	return c.archiveReader
}
