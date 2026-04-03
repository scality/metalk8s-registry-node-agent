//nolint:dupl // Duplication is fine in DI packages.
package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/usecase"

func (c *Container) getUploadPartUseCase() *usecase.UploadPart {
	if c.uploadPartUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.uploadPartUseCase = usecase.NewUploadPart(
			c.GetLogger(),
			store,
			c.GetFileSystemBucketManager(),
			store,
			c.GetFileSystemMultipartRemover(),
			c.GetFileSystemMultipartInspector(),
			c.GetRootExternAPIPath(),
		)
	}

	return c.uploadPartUseCase
}

func (c *Container) GetInitializeSessionUseCase() *usecase.InitializeSession {
	if c.initializeSessionUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.initializeSessionUseCase = usecase.NewInitializeSession(
			c.GetLogger(),
			store,
			c.GetFileSystemBucketManager(),
			c.GetFileSystemArchiveLister(),
			store,
			c.GetFileSystemMultipartInspector(),
		)
	}

	return c.initializeSessionUseCase
}

func (c *Container) GetRemoveSolutionArchiveUseCase() *usecase.RemoveSolutionArchive {
	if c.removeSolutionArchiveUseCase == nil {
		c.removeSolutionArchiveUseCase = usecase.NewRemoveSolutionArchive(
			c.GetLogger(),
			c.GetFSSolutionArchiveStorage(),
			c.GetFileSystemArchiveLister(),
			c.GetFileSystemArchiveRemover(),
		)
	}
	return c.removeSolutionArchiveUseCase
}

func (c *Container) GetValidateSolutionArchiveUseCase() *usecase.ValidateSolutionArchive {
	if c.validateSolutionArchiveUseCase == nil {
		c.validateSolutionArchiveUseCase = usecase.NewValidateSolutionArchive(
			c.GetLogger(),
			c.GetFSSolutionArchiveStorage(),
			c.GetFileSystemArchiveLister(),
			c.GetFileSystemArchiveRemover(),
		)
	}
	return c.validateSolutionArchiveUseCase
}

func (c *Container) GetRemoveSessionUseCase() *usecase.RemoveSession {
	if c.removeSessionUseCase == nil {
		c.removeSessionUseCase = usecase.NewRemoveSession(
			c.GetLogger(),
			c.GetFSSolutionArchiveStorage(),
			c.GetFileSystemBucketManager(),
		)
	}
	return c.removeSessionUseCase
}

func (c *Container) GetDownloadSolutionArchiveUseCase() *usecase.DownloadSolutionArchive {
	if c.downloadSolutionArchiveUseCase == nil {
		c.downloadSolutionArchiveUseCase = usecase.NewDownloadSolutionArchive(
			c.GetLogger(),
			c.GetFSSolutionArchiveStorage(),
			c.GetFileSystemArchiveLister(),
			c.GetFileSystemArchiveReader(),
			c.GetRootInternAPIPath(),
		)
	}
	return c.downloadSolutionArchiveUseCase
}

func (c *Container) GetGetExternalSolutionArchiveUseCase() *usecase.GetExternalSolutionArchive {
	if c.getExternalSolutionArchiveUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.getExternalSolutionArchiveUseCase = usecase.NewGetExternalSolutionArchive(
			c.GetLogger(),
			store,
			c.getHTTPExternalDownloader(),
			c.GetFileSystemArchiveLister(),
			store,
		)
	}
	return c.getExternalSolutionArchiveUseCase
}

func (c *Container) GetMountSolutionArchiveUseCase() *usecase.MountSolutionArchive {
	if c.mountSolutionArchiveUseCase == nil {
		c.mountSolutionArchiveUseCase = usecase.NewMountSolutionArchive(
			c.GetLogger(),
			c.GetFSSolutionArchiveStorage(),
			c.GetFileSystemArchiveMounter(),
			c.GetFileSystemArchiveLister(),
		)
	}
	return c.mountSolutionArchiveUseCase
}

func (c *Container) GetUnmountSolutionArchiveUseCase() *usecase.UnmountSolutionArchive {
	if c.unmountSolutionArchiveUseCase == nil {
		c.unmountSolutionArchiveUseCase = usecase.NewUnmountSolutionArchive(
			c.GetLogger(),
			c.GetFSSolutionArchiveStorage(),
			c.GetFileSystemArchiveMounter(),
		)
	}
	return c.unmountSolutionArchiveUseCase
}

func (c *Container) GetCleanArchivesUseCase() *usecase.CleanArchive {
	if c.cleanArchivesUseCase == nil {
		c.cleanArchivesUseCase = usecase.NewCleanArchive(
			c.GetLogger(),
			c.GetFileSystemArchiveCleaner(),
			c.GetFileSystemFileWatcher(),
			c.deleteChan,
		)
	}
	return c.cleanArchivesUseCase
}
