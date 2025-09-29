package di

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/generated"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/resolver"
)

//nolint:ireturn
func (c *Container) getHTTPResolver() generated.StrictServerInterface {
	if c.resolver == nil {
		c.resolver = resolver.NewRoot(
			c.getUploadPartHandler(),
		)
	}

	return c.resolver
}
