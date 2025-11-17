package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/partuploader"
)

func (c *Container) getStoragePartUploader() *partuploader.Storage {
	if c.storagePartUploader == nil {
		c.storagePartUploader = partuploader.NewStorage(
			c.GetFileSystemArtifactStorage(),
			c.GetLogger(),
			c.GetRootExternAPIPath(),
		)
	}

	return c.storagePartUploader
}
