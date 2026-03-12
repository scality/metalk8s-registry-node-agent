package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/externalsolutionarchivegetter"

func (c *Container) getStorageExternalSolutionArchiveGetter() *externalsolutionarchivegetter.Storage {
	if c.storageExternalSolutionArchiveGetter == nil {
		c.storageExternalSolutionArchiveGetter = externalsolutionarchivegetter.NewStorage(
			c.GetFSSolutionArchiveStorage(),
			c.GetLogger(),
			c.getHTTPExternalDownloader(),
			c.GetRootExternAPIPath(),
			c.GetChunkSize(),
		)
	}

	return c.storageExternalSolutionArchiveGetter
}
