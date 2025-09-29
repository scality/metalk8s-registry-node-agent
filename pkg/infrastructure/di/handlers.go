//nolint:dupl // Duplication is fine in DI packages.
package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/handler"

func (c *Container) getUploadPartHandler() *handler.UploadPart {
	if c.uploadPartHandler == nil {
		c.uploadPartHandler = handler.NewUploadPart(
			c.GetLogger(),
			c.getUploadPartUseCase(),
		)
	}

	return c.uploadPartHandler
}
