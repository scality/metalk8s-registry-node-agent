//nolint:dupl // Duplication is fine in DI packages.
package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/usecase"

func (c *Container) getUploadPartUseCase() *usecase.UploadPart {
	if c.uploadPartUseCase == nil {
		c.uploadPartUseCase = usecase.NewUploadPart(
			c.GetLogger(),
			c.getStoragePartUploader(),
		)
	}

	return c.uploadPartUseCase
}

func (c *Container) GetInitializeSessionUseCase() *usecase.InitializeSession {
	if c.initializeSessionUseCase == nil {
		c.initializeSessionUseCase = usecase.NewInitializeSession(
			c.GetLogger(),
			c.getStorageSessionInitializer(),
		)
	}

	return c.initializeSessionUseCase
}

func (c *Container) GetRemoveSolutionArchiveUseCase() *usecase.RemoveSolutionArchive {
	if c.removeSolutionArchiveUseCase == nil {
		c.removeSolutionArchiveUseCase = usecase.NewRemoveSolutionArchive(
			c.GetLogger(),
			c.getStorageSolutionArchiveRemover(),
		)
	}
	return c.removeSolutionArchiveUseCase
}

func (c *Container) GetValidateSolutionArchiveUseCase() *usecase.ValidateSolutionArchive {
	if c.validateSolutionArchiveUseCase == nil {
		c.validateSolutionArchiveUseCase = usecase.NewValidateSolutionArchive(
			c.GetLogger(),
			c.getStorageSolutionArchiveValidator(),
		)
	}
	return c.validateSolutionArchiveUseCase
}

func (c *Container) GetRemoveSessionUseCase() *usecase.RemoveSession {
	if c.removeSessionUseCase == nil {
		c.removeSessionUseCase = usecase.NewRemoveSession(
			c.GetLogger(),
			c.getStorageSessionRemover(),
		)
	}
	return c.removeSessionUseCase
}

func (c *Container) GetDownloadSolutionArchiveUseCase() *usecase.DownloadSolutionArchive {
	if c.downloadSolutionArchiveUseCase == nil {
		c.downloadSolutionArchiveUseCase = usecase.NewDownloadSolutionArchive(
			c.GetLogger(),
			c.getStorageSolutionArchiveDownloader(),
		)
	}
	return c.downloadSolutionArchiveUseCase
}

func (c *Container) GetGetExternalSolutionArchiveUseCase() *usecase.GetExternalSolutionArchive {
	if c.getExternalSolutionArchiveUseCase == nil {
		c.getExternalSolutionArchiveUseCase = usecase.NewGetExternalSolutionArchive(
			c.GetLogger(),
			c.getStorageExternalSolutionArchiveGetter(),
		)
	}
	return c.getExternalSolutionArchiveUseCase
}

func (c *Container) GetMountSolutionArchiveUseCase() *usecase.MountSolutionArchive {
	if c.mountSolutionArchiveUseCase == nil {
		c.mountSolutionArchiveUseCase = usecase.NewMountSolutionArchive(
			c.GetLogger(),
			c.getStorageSolutionArchiveMounter(),
		)
	}
	return c.mountSolutionArchiveUseCase
}

func (c *Container) GetUnmountSolutionArchiveUseCase() *usecase.UnmountSolutionArchive {
	if c.unmountSolutionArchiveUseCase == nil {
		c.unmountSolutionArchiveUseCase = usecase.NewUnmountSolutionArchive(
			c.GetLogger(),
			c.getStorageSolutionArchiveUnmounter(),
		)
	}
	return c.unmountSolutionArchiveUseCase
}

func (c *Container) GetCleanUnusedSolutionArchivesUseCase() *usecase.CleanUnusedSolutionArchives {
	if c.cleanUnusedSolutionArchivesUseCase == nil {
		c.cleanUnusedSolutionArchivesUseCase = usecase.NewCleanUnusedSolutionArchives(
			c.GetLogger(),
			c.getStorageSolutionArchiveCleaner(),
		)
	}
	return c.cleanUnusedSolutionArchivesUseCase
}

func (c *Container) GetCleanUnusedSolutionsUseCase() *usecase.CleanUnusedSolutions {
	if c.cleanUnusedSolutionsUseCase == nil {
		c.cleanUnusedSolutionsUseCase = usecase.NewCleanUnusedSolutions(
			c.GetLogger(),
			c.getStorageSolutionCleaner(),
		)
	}
	return c.cleanUnusedSolutionsUseCase
}
