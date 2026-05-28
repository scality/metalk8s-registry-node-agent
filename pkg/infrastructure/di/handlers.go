//nolint:dupl // Duplication is fine in DI packages.
package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/handler"

func (c *Container) getUploadPartHandler() *handler.UploadPart {
	if c.uploadPartHandler == nil {
		c.uploadPartHandler = handler.NewUploadPart(
			c.GetLogger(),
			c.getReceivePartUseCase(),
		)
	}

	return c.uploadPartHandler
}

func (c *Container) getDownloadSolutionArchiveHandler() *handler.DownloadSolutionArchive {
	if c.downloadSolutionArchiveHandler == nil {
		c.downloadSolutionArchiveHandler = handler.NewDownloadSolutionArchive(
			c.GetLogger(),
			c.GetServePartUseCase(),
		)
	}
	return c.downloadSolutionArchiveHandler
}

func (c *Container) getDescribeSolutionArchiveHandler() *handler.DescribeSolutionArchive {
	if c.describeSolutionArchiveHandler == nil {
		c.describeSolutionArchiveHandler = handler.NewDescribeSolutionArchive(
			c.GetLogger(),
			c.GetDescribeSolutionArchiveUseCase(),
		)
	}
	return c.describeSolutionArchiveHandler
}
