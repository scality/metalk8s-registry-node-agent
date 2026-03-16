//nolint:dupl // Duplication is fine in DI packages.
package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/usecase"

func (c *Container) getUploadPartUseCase() *usecase.UploadPart {
	if c.uploadPartUseCase == nil {
		c.uploadPartUseCase = usecase.NewUploadPart(
			c.GetLogger(),
			c.GetFileSystemBucketManager(),
			c.GetFileSystemMultipartUploader(),
			c.GetFileSystemMultipartRemover(),
			c.GetFileSystemMultipartInspector(),
			c.getInMemoryBucketLocker(),
			c.getInMemoryArchiveLocker(),
			c.GetRootExternAPIPath(),
		)
	}

	return c.uploadPartUseCase
}

func (c *Container) GetInitializeSessionUseCase() *usecase.InitializeSession {
	if c.initializeSessionUseCase == nil {
		c.initializeSessionUseCase = usecase.NewInitializeSession(
			c.GetLogger(),
			c.GetFileSystemBucketManager(),
			c.GetFileSystemArchiveLister(),
			c.GetFileSystemMultipartUploader(),
			c.GetFileSystemMultipartInspector(),
			c.getInMemoryBucketLocker(),
		)
	}

	return c.initializeSessionUseCase
}

func (c *Container) GetRemoveSolutionArchiveUseCase() *usecase.RemoveSolutionArchive {
	if c.removeSolutionArchiveUseCase == nil {
		c.removeSolutionArchiveUseCase = usecase.NewRemoveSolutionArchive(
			c.GetLogger(),
			c.GetFileSystemArchiveLister(),
			c.GetFileSystemArchiveRemover(),
			c.getInMemoryArchiveLocker(),
		)
	}
	return c.removeSolutionArchiveUseCase
}

func (c *Container) GetValidateSolutionArchiveUseCase() *usecase.ValidateSolutionArchive {
	if c.validateSolutionArchiveUseCase == nil {
		c.validateSolutionArchiveUseCase = usecase.NewValidateSolutionArchive(
			c.GetLogger(),
			c.GetFileSystemArchiveLister(),
			c.GetFileSystemArchiveRemover(),
			c.getInMemoryArchiveLocker(),
		)
	}
	return c.validateSolutionArchiveUseCase
}

func (c *Container) GetRemoveSessionUseCase() *usecase.RemoveSession {
	if c.removeSessionUseCase == nil {
		c.removeSessionUseCase = usecase.NewRemoveSession(
			c.GetLogger(),
			c.GetFileSystemBucketManager(),
			c.getInMemoryBucketLocker(),
		)
	}
	return c.removeSessionUseCase
}

func (c *Container) GetDownloadSolutionArchiveUseCase() *usecase.DownloadSolutionArchive {
	if c.downloadSolutionArchiveUseCase == nil {
		c.downloadSolutionArchiveUseCase = usecase.NewDownloadSolutionArchive(
			c.GetLogger(),
			c.GetFileSystemArchiveLister(),
			c.GetFileSystemArchiveReader(),
			c.getInMemoryArchiveLocker(),
			c.GetRootInternAPIPath(),
		)
	}
	return c.downloadSolutionArchiveUseCase
}

func (c *Container) GetDescribeSolutionArchiveUseCase() *usecase.DescribeSolutionArchive {
	if c.describeSolutionArchiveUseCase == nil {
		c.describeSolutionArchiveUseCase = usecase.NewDescribeSolutionArchive(
			c.GetLogger(),
			c.GetFileSystemArchiveLister(),
			c.getInMemoryArchiveLocker(),
			c.GetRootInternAPIPath(),
		)
	}
	return c.describeSolutionArchiveUseCase
}

func (c *Container) GetGetExternalSolutionArchiveUseCase() *usecase.GetExternalSolutionArchive {
	if c.getExternalSolutionArchiveUseCase == nil {
		c.getExternalSolutionArchiveUseCase = usecase.NewGetExternalSolutionArchive(
			c.GetLogger(),
			c.getHTTPExternalDownloader(),
			c.GetFileSystemBucketManager(),
			c.GetFileSystemArchiveLister(),
			c.getInMemoryArchiveLocker(),
			c.getInMemoryBucketLocker(),
			c.GetFileSystemMultipartUploader(),
			c.GetFileSystemMultipartInspector(),
			c.GetFileSystemMultipartRemover(),
			c.GetRootExternAPIPath(),
			c.GetChunkSize(),
		)
	}
	return c.getExternalSolutionArchiveUseCase
}

func (c *Container) GetMountSolutionArchiveUseCase() *usecase.MountSolutionArchive {
	if c.mountSolutionArchiveUseCase == nil {
		c.mountSolutionArchiveUseCase = usecase.NewMountSolutionArchive(
			c.GetLogger(),
			c.GetFileSystemArchiveMounter(),
			c.GetFileSystemArchiveLister(),
			c.getInMemoryArchiveLocker(),
		)
	}
	return c.mountSolutionArchiveUseCase
}

func (c *Container) GetUnmountSolutionArchiveUseCase() *usecase.UnmountSolutionArchive {
	if c.unmountSolutionArchiveUseCase == nil {
		c.unmountSolutionArchiveUseCase = usecase.NewUnmountSolutionArchive(
			c.GetLogger(),
			c.GetFileSystemArchiveMounter(),
			c.getInMemoryArchiveLocker(),
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
