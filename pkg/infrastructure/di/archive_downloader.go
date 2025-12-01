package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivedownloader"

func (c *Container) getStorageSolutionArchiveDownloader() *archivedownloader.Storage {
	if c.storageSolutionArchiveDownloader == nil {
		c.storageSolutionArchiveDownloader = archivedownloader.NewStorage(
			c.GetFSSolutionArchiveStorage(),
			c.GetLogger(),
			c.GetRootInternAPIPath(),
		)
	}

	return c.storageSolutionArchiveDownloader
}
