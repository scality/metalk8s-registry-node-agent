package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/resolver"
)

//nolint:ireturn
func (c *Container) getHTTPResolver() extern.StrictServerInterface {
	if c.externResolver == nil {
		c.externResolver = resolver.NewExternRoot(
			c.getUploadPartHandler(),
		)
	}

	return c.externResolver
}
