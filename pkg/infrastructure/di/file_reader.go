package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/filereader"

func (c *Container) GetFilesystemFileReader() *filereader.FileSystem {
	if c.filesystemFileReader == nil {
		lifecycle := c.GetFilesystemStorageLifecycle()
		c.filesystemFileReader = filereader.NewFileSystem(
			c.GetLogger(),
			lifecycle.SolutionArchivesLocation(),
		)
	}

	return c.filesystemFileReader
}
