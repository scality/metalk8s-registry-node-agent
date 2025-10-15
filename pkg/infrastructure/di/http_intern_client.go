// nolint: dupl // normal to have the internal and external clients very similar
package di

import (
	"crypto/tls"
	"net/http"

	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
)

func (c *Container) GetHTTPInternClient() *http.Client {
	if c.httpInternClient == nil {
		c.httpInternClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// nolint: gosec // TODO: Certificate validation.
					InsecureSkipVerify: true,
				},
			},
		}
	}

	return c.httpInternClient
}

func (c *Container) GetGeneratedHTTPInternClient() *intern.ClientWithResponses {
	if c.generatedHTTPInternClient == nil {
		client, err := intern.NewClientWithResponses(
			"http://localhost"+c.config.Intern.Addr+c.GetRootInternAPIPath(),
			intern.WithHTTPClient(c.GetHTTPInternClient()),
		)
		if err != nil {
			c.GetLogger().Fatal().Err(err).Msg("failed to create generated http client")
		}

		c.generatedHTTPInternClient = client
	}

	return c.generatedHTTPInternClient
}
