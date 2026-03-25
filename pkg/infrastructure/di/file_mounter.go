package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/filemounter"

func (c *Container) GetFilesystemFileMounter() *filemounter.FileSystem {
	if c.filesystemFileMounter == nil {
		lifecycle := c.GetStorageLifecycle()
		c.filesystemFileMounter = filemounter.NewFileSystem(
			c.GetLogger(),
			lifecycle.SolutionArchivesLocation(),
			lifecycle.SolutionsLocation(),
			c.GetFilesystemFileRemover(),
			c.GetFilesystemFileWatcher(),
		)
	}

	return c.filesystemFileMounter
}
