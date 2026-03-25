package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/filewatcher"

func (c *Container) GetFilesystemFileWatcher() *filewatcher.FileSystem {
	if c.filesystemFileWatcher == nil {
		lifecycle := c.GetFilesystemStorageLifecycle()
		c.filesystemFileWatcher = filewatcher.NewFileSystem(lifecycle.Watcher())
	}

	return c.filesystemFileWatcher
}
