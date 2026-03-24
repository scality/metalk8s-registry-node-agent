//nolint:dupl // Duplication is fine in DI packages.
package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/usecase"

func (c *Container) getUploadPartUseCase() *usecase.UploadPart {
	if c.uploadPartUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.uploadPartUseCase = usecase.NewUploadPart(
			c.GetLogger(),
			store,
			store,
			store,
			store,
			c.rootInternAPIPath,
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
			store,
			store,
			store,
			store,
		)
	}

	return c.initializeSessionUseCase
}

func (c *Container) GetRemoveSolutionArchiveUseCase() *usecase.RemoveSolutionArchive {
	if c.removeSolutionArchiveUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.removeSolutionArchiveUseCase = usecase.NewRemoveSolutionArchive(
			c.GetLogger(),
			store,
			store,
			store,
		)
	}
	return c.removeSolutionArchiveUseCase
}

func (c *Container) GetValidateSolutionArchiveUseCase() *usecase.ValidateSolutionArchive {
	if c.validateSolutionArchiveUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.validateSolutionArchiveUseCase = usecase.NewValidateSolutionArchive(
			c.GetLogger(),
			store,
			store,
			store,
		)
	}
	return c.validateSolutionArchiveUseCase
}

func (c *Container) GetRemoveSessionUseCase() *usecase.RemoveSession {
	if c.removeSessionUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.removeSessionUseCase = usecase.NewRemoveSession(
			c.GetLogger(),
			store,
			store,
		)
	}
	return c.removeSessionUseCase
}

func (c *Container) GetDownloadSolutionArchiveUseCase() *usecase.DownloadSolutionArchive {
	if c.downloadSolutionArchiveUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.downloadSolutionArchiveUseCase = usecase.NewDownloadSolutionArchive(
			c.GetLogger(),
			store,
			store,
			store,
			c.rootInternAPIPath,
		)
	}
	return c.downloadSolutionArchiveUseCase
}

func (c *Container) GetDescribeSolutionArchiveUseCase() *usecase.DescribeSolutionArchive {
	if c.describeSolutionArchiveUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.describeSolutionArchiveUseCase = usecase.NewDescribeSolutionArchive(
			c.GetLogger(),
			store,
			store,
			c.rootInternAPIPath,
		)
	}
	return c.describeSolutionArchiveUseCase
}

func (c *Container) GetGetExternalSolutionArchiveUseCase() *usecase.GetExternalSolutionArchive {
	if c.getExternalSolutionArchiveUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.getExternalSolutionArchiveUseCase = usecase.NewGetExternalSolutionArchive(
			store,
			store,
			store,
			store,
			store,
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
		store := c.GetFSSolutionArchiveStorage()
		c.mountSolutionArchiveUseCase = usecase.NewMountSolutionArchive(
			c.GetLogger(),
			store,
			store,
			store,
		)
	}
	return c.mountSolutionArchiveUseCase
}

func (c *Container) GetUnmountSolutionArchiveUseCase() *usecase.UnmountSolutionArchive {
	if c.unmountSolutionArchiveUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.unmountSolutionArchiveUseCase = usecase.NewUnmountSolutionArchive(
			c.GetLogger(),
			store,
			store,
		)
	}
	return c.unmountSolutionArchiveUseCase
}

func (c *Container) GetCleanSolutionArchiveUseCase() *usecase.CleanSolutionArchive {
	if c.cleanSolutionArchiveUseCase == nil {
		store := c.GetFSSolutionArchiveStorage()
		c.cleanSolutionArchiveUseCase = usecase.NewCleanSolutionArchive(
			c.GetLogger(),
			store,
			store,
			store,
			c.deleteChan,
		)
	}

	return c.cleanSolutionArchiveUseCase
}
