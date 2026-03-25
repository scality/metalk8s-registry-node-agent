package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/filelister"

func (c *Container) GetFilesystemFileLister() *filelister.FileSystem {
	if c.filesystemFileLister == nil {
		lifecycle := c.GetStorageLifecycle()
		c.filesystemFileLister = filelister.NewFileSystem(
			c.GetLogger(),
			lifecycle.SolutionArchivesLocation(),
			lifecycle.InterestContentFilter(),
			lifecycle.WatchedFileStore(),
		)
	}

	return c.filesystemFileLister
}
