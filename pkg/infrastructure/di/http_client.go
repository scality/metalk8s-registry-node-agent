package di

import (
	"crypto/tls"
	"net/http"

	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/generated"
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

func (c *Container) GetGeneratedHTTPClient() *generated.ClientWithResponses {
	if c.generatedHTTPClient == nil {
		client, err := generated.NewClientWithResponses(
			"http://localhost:5001"+c.GetRootAPIPath(),
			generated.WithHTTPClient(c.GetHTTPClient()),
		)
		if err != nil {
			c.GetLogger().Fatal().Err(err).Msg("failed to create generated http client")
		}

		c.generatedHTTPClient = client
	}

	return c.generatedHTTPClient
}
