package di

import (
	"crypto/tls"
	"net/http"

	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
)

func (c *Container) GetHTTPExternClient() *http.Client {
	if c.httpExternClient == nil {
		c.httpExternClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// nolint: gosec // TODO: Certificate validation.
					InsecureSkipVerify: true,
				},
			},
		}
	}

	return c.httpExternClient
}

func (c *Container) GetGeneratedHTTPExternClient() *extern.ClientWithResponses {
	if c.generatedHTTPExternClient == nil {
		client, err := extern.NewClientWithResponses(
			"http://localhost:5001"+c.GetRootExternAPIPath(),
			extern.WithHTTPClient(c.GetHTTPExternClient()),
		)
		if err != nil {
			c.GetLogger().Fatal().Err(err).Msg("failed to create generated http client")
		}

		c.generatedHTTPExternClient = client
	}

	return c.generatedHTTPExternClient
}
