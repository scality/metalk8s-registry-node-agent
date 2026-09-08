package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/mountwatcher"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

func (c *Container) GetFileSystemMountWatcher() service.MountWatcher {
	if c.mountWatcher == nil {
		c.mountWatcher = mountwatcher.NewFileSystem(
			c.GetLogger(),
			c.config.SolutionsLocation,
			c.config.MountWatcherPollTimeoutMS,
		)
	}

	return c.mountWatcher
}
