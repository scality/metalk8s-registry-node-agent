package di

import (
	"crypto/tls"
	"net/http"

	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
)

func (c *Container) GetHTTPClient() *http.Client {
	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// nolint: gosec // TODO: Certificate validation.
					InsecureSkipVerify: true,
				},
			},
		}
	}

	return c.httpClient
}

func (c *Container) GetGeneratedHTTPClient() *extern.ClientWithResponses {
	if c.generatedHTTPClient == nil {
		client, err := extern.NewClientWithResponses(
			"http://localhost:5001"+c.GetRootAPIPath(),
			extern.WithHTTPClient(c.GetHTTPClient()),
		)
		if err != nil {
			c.GetLogger().Fatal().Err(err).Msg("failed to create generated http client")
		}

		c.generatedHTTPClient = client
	}

	return c.generatedHTTPClient
}
