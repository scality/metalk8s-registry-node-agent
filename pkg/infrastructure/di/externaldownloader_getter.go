package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/externaldownloader"

func (c *Container) getHTTPExternalDownloader() *externaldownloader.HTTP {
	if c.httpExternalDownloader == nil {
		c.httpExternalDownloader = externaldownloader.NewHTTP(
			c.GetLogger(),
			c.GetHTTPInternClient(),
		)
	}
	return c.httpExternalDownloader
}
