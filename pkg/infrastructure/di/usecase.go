//nolint:dupl // Duplication is fine in DI packages.
package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/usecase"

func (c *Container) getReceivePartUseCase() *usecase.ReceivePart {
	if c.receivePartUseCase == nil {
		c.receivePartUseCase = usecase.NewReceivePart(
			c.GetLogger(),
			c.getFileSystemBucketManager(),
			c.getFileSystemMultipartStorer(),
			c.getFileSystemMultipartInspector(),
			c.getInMemoryBucketLocker(),
			c.getInMemoryArchiveLocker(),
			c.getRootExternAPIPath(),
		)
	}

	return c.receivePartUseCase
}

func (c *Container) GetInitializeSessionUseCase() *usecase.InitializeSession {
	if c.initializeSessionUseCase == nil {
		c.initializeSessionUseCase = usecase.NewInitializeSession(
			c.GetLogger(),
			c.getFileSystemBucketManager(),
			c.getFileSystemArchiveLister(),
			c.getFileSystemMultipartStorer(),
			c.getFileSystemMultipartInspector(),
			c.getInMemoryBucketLocker(),
		)
	}

	return c.initializeSessionUseCase
}

func (c *Container) GetRemoveSolutionArchiveUseCase() *usecase.RemoveSolutionArchive {
	if c.removeSolutionArchiveUseCase == nil {
		c.removeSolutionArchiveUseCase = usecase.NewRemoveSolutionArchive(
			c.GetLogger(),
			c.getFileSystemArchiveLister(),
			c.getFileSystemArchiveRemover(),
			c.getInMemoryArchiveLocker(),
		)
	}
	return c.removeSolutionArchiveUseCase
}

func (c *Container) GetValidateSolutionArchiveUseCase() *usecase.ValidateSolutionArchive {
	if c.validateSolutionArchiveUseCase == nil {
		c.validateSolutionArchiveUseCase = usecase.NewValidateSolutionArchive(
			c.GetLogger(),
			c.getFileSystemArchiveLister(),
			c.getFileSystemArchiveRemover(),
			c.getInMemoryArchiveLocker(),
			c.GetMetricsRecorder(),
		)
	}
	return c.validateSolutionArchiveUseCase
}

func (c *Container) GetRemoveSessionUseCase() *usecase.RemoveSession {
	if c.removeSessionUseCase == nil {
		c.removeSessionUseCase = usecase.NewRemoveSession(
			c.GetLogger(),
			c.getFileSystemBucketManager(),
			c.getInMemoryBucketLocker(),
		)
	}
	return c.removeSessionUseCase
}

func (c *Container) getServePartUseCase() *usecase.ServePart {
	if c.servePartUseCase == nil {
		c.servePartUseCase = usecase.NewServePart(
			c.GetLogger(),
			c.getFileSystemArchiveLister(),
			c.getFileSystemArchiveReader(),
			c.getInMemoryArchiveLocker(),
			c.GetRootInternAPIPath(),
		)
	}
	return c.servePartUseCase
}

func (c *Container) getDescribeSolutionArchiveUseCase() *usecase.DescribeSolutionArchive {
	if c.describeSolutionArchiveUseCase == nil {
		c.describeSolutionArchiveUseCase = usecase.NewDescribeSolutionArchive(
			c.GetLogger(),
			c.getFileSystemArchiveLister(),
			c.getInMemoryArchiveLocker(),
			c.GetRootInternAPIPath(),
		)
	}
	return c.describeSolutionArchiveUseCase
}

func (c *Container) GetDownloadPartUseCase() *usecase.DownloadPart {
	if c.downloadPartUseCase == nil {
		c.downloadPartUseCase = usecase.NewDownloadPart(
			c.GetLogger(),
			c.getHTTPExternalDownloader(),
			c.getFileSystemBucketManager(),
			c.getFileSystemArchiveLister(),
			c.getInMemoryArchiveLocker(),
			c.getInMemoryBucketLocker(),
			c.getFileSystemMultipartStorer(),
			c.getFileSystemMultipartInspector(),
			c.getRootExternAPIPath(),
			c.getChunkSize(),
			c.GetMetricsRecorder(),
		)
	}
	return c.downloadPartUseCase
}

func (c *Container) GetMountSolutionArchiveUseCase() *usecase.MountSolutionArchive {
	if c.mountSolutionArchiveUseCase == nil {
		c.mountSolutionArchiveUseCase = usecase.NewMountSolutionArchive(
			c.GetLogger(),
			c.getFileSystemArchiveMounter(),
			c.getFileSystemArchiveLister(),
			c.getInMemoryArchiveLocker(),
		)
	}
	return c.mountSolutionArchiveUseCase
}

func (c *Container) GetUnmountSolutionArchiveUseCase() *usecase.UnmountSolutionArchive {
	if c.unmountSolutionArchiveUseCase == nil {
		c.unmountSolutionArchiveUseCase = usecase.NewUnmountSolutionArchive(
			c.GetLogger(),
			c.getFileSystemArchiveMounter(),
			c.getInMemoryArchiveLocker(),
		)
	}
	return c.unmountSolutionArchiveUseCase
}

func (c *Container) GetCleanArchivesUseCase() *usecase.CleanArchive {
	if c.cleanArchivesUseCase == nil {
		c.cleanArchivesUseCase = usecase.NewCleanArchive(
			c.GetLogger(),
			c.getFileSystemArchiveCleaner(),
			c.GetFileSystemFileWatcher(),
			c.deleteChan,
		)
	}
	return c.cleanArchivesUseCase
}
