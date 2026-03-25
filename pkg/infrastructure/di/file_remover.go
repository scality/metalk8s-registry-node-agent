package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/fileremover"

func (c *Container) GetFilesystemFileRemover() *fileremover.FileSystem {
	if c.filesystemFileRemover == nil {
		lifecycle := c.GetStorageLifecycle()
		c.filesystemFileRemover = fileremover.NewFileSystem(
			c.GetLogger(),
			lifecycle.SolutionArchivesLocation(),
			lifecycle.WatchedFileStore(),
		)
	}

	return c.filesystemFileRemover
}
