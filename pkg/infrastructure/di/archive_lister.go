package di

import (
	archivelister "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivelister"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) getFileSystemArchiveLister() service.ArchiveLister {
	if c.archiveLister == nil {
		c.archiveLister = archivelister.NewFileSystem(
			c.GetLogger(),
			c.config.SolutionArchivesLocation,
			library.NewRegexNormalFileFilter(solutionArchiveStorageNameRegexp),
			c.GetFSSolutionArchiveStorage(),
		)
	}
	return c.archiveLister
}

func (c *Container) GetMockFileSystemArchiveLister() service.ArchiveLister {
	if c.archiveLister == nil {
		c.archiveLister = c.GetMockFSSolutionArchiveStorage().(service.ArchiveLister)
	}
	return c.archiveLister
}
