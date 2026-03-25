//nolint:dupl // Duplication is fine in DI packages.
package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/usecase"

func (c *Container) getUploadPartUseCase() *usecase.UploadPart {
	if c.uploadPartUseCase == nil {
		c.uploadPartUseCase = usecase.NewUploadPart(
			c.GetLogger(),
			&c.storageMu,
			c.GetFilesystemBucketManager(),
			c.GetFilesystemMultipartUploader(),
			c.GetFilesystemMultipartInspector(),
			c.rootInternAPIPath,
		)
	}

	return c.uploadPartUseCase
}

func (c *Container) GetInitializeSessionUseCase() *usecase.InitializeSession {
	if c.initializeSessionUseCase == nil {
		c.initializeSessionUseCase = usecase.NewInitializeSession(
			c.GetLogger(),
			&c.storageMu,
			c.GetFilesystemFileLister(),
			c.GetFilesystemBucketManager(),
			c.GetFilesystemMultipartUploader(),
			c.GetFilesystemMultipartInspector(),
		)
	}

	return c.initializeSessionUseCase
}

func (c *Container) GetRemoveSolutionArchiveUseCase() *usecase.RemoveSolutionArchive {
	if c.removeSolutionArchiveUseCase == nil {
		c.removeSolutionArchiveUseCase = usecase.NewRemoveSolutionArchive(
			c.GetLogger(),
			&c.storageMu,
			c.GetFilesystemFileLister(),
			c.GetFilesystemFileRemover(),
		)
	}
	return c.removeSolutionArchiveUseCase
}

func (c *Container) GetValidateSolutionArchiveUseCase() *usecase.ValidateSolutionArchive {
	if c.validateSolutionArchiveUseCase == nil {
		c.validateSolutionArchiveUseCase = usecase.NewValidateSolutionArchive(
			c.GetLogger(),
			&c.storageMu,
			c.GetFilesystemFileLister(),
			c.GetFilesystemFileRemover(),
		)
	}
	return c.validateSolutionArchiveUseCase
}

func (c *Container) GetRemoveSessionUseCase() *usecase.RemoveSession {
	if c.removeSessionUseCase == nil {
		c.removeSessionUseCase = usecase.NewRemoveSession(
			c.GetLogger(),
			&c.storageMu,
			c.GetFilesystemBucketManager(),
		)
	}
	return c.removeSessionUseCase
}

func (c *Container) GetDownloadSolutionArchiveUseCase() *usecase.DownloadSolutionArchive {
	if c.downloadSolutionArchiveUseCase == nil {
		c.downloadSolutionArchiveUseCase = usecase.NewDownloadSolutionArchive(
			c.GetLogger(),
			&c.storageMu,
			c.GetFilesystemFileLister(),
			c.GetFilesystemFileReader(),
			c.rootInternAPIPath,
		)
	}
	return c.downloadSolutionArchiveUseCase
}

func (c *Container) GetDescribeSolutionArchiveUseCase() *usecase.DescribeSolutionArchive {
	if c.describeSolutionArchiveUseCase == nil {
		c.describeSolutionArchiveUseCase = usecase.NewDescribeSolutionArchive(
			c.GetLogger(),
			&c.storageMu,
			c.GetFilesystemFileLister(),
			c.rootInternAPIPath,
		)
	}
	return c.describeSolutionArchiveUseCase
}

func (c *Container) GetGetExternalSolutionArchiveUseCase() *usecase.GetExternalSolutionArchive {
	if c.getExternalSolutionArchiveUseCase == nil {
		c.getExternalSolutionArchiveUseCase = usecase.NewGetExternalSolutionArchive(
			&c.storageMu,
			c.GetFilesystemFileLister(),
			c.GetFilesystemBucketManager(),
			c.GetFilesystemMultipartUploader(),
			c.GetFilesystemMultipartInspector(),
			c.GetLogger(),
			c.getHTTPExternalDownloader(),
			c.rootInternAPIPath,
			c.chunkSize,
		)
	}

	return c.getExternalSolutionArchiveUseCase
}

func (c *Container) GetMountSolutionArchiveUseCase() *usecase.MountSolutionArchive {
	if c.mountSolutionArchiveUseCase == nil {
		c.mountSolutionArchiveUseCase = usecase.NewMountSolutionArchive(
			c.GetLogger(),
			&c.storageMu,
			c.GetFilesystemFileLister(),
			c.GetFilesystemFileMounter(),
		)
	}
	return c.mountSolutionArchiveUseCase
}

func (c *Container) GetUnmountSolutionArchiveUseCase() *usecase.UnmountSolutionArchive {
	if c.unmountSolutionArchiveUseCase == nil {
		c.unmountSolutionArchiveUseCase = usecase.NewUnmountSolutionArchive(
			c.GetLogger(),
			&c.storageMu,
			c.GetFilesystemFileMounter(),
		)
	}
	return c.unmountSolutionArchiveUseCase
}

func (c *Container) GetCleanSolutionArchiveUseCase() *usecase.CleanSolutionArchive {
	if c.cleanSolutionArchiveUseCase == nil {
		c.cleanSolutionArchiveUseCase = usecase.NewCleanSolutionArchive(
			c.GetLogger(),
			&c.storageMu,
			c.GetFilesystemFileMounter(),
			c.GetFilesystemFileWatcher(),
			c.deleteChan,
		)
	}

	return c.cleanSolutionArchiveUseCase
}
