// nolint: dupl // normal to have the internal and external clients very similar
package di

import (
	"bytes"
	"io"
	"net/http"

	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
)

func (c *Container) GetHTTPInternClient() *http.Client {
	if c.httpInternClient == nil {
		c.httpInternClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: c.getInternTLSClientConfig(),
			},
		}
	}

	return c.httpInternClient
}

func (c *Container) GetGeneratedHTTPInternClient() *intern.ClientWithResponses {
	if c.generatedHTTPInternClient == nil {
		client, err := intern.NewClientWithResponses(
			"https://localhost"+c.config.Intern.Addr+c.rootInternAPIPath,
			intern.WithHTTPClient(c.GetHTTPInternClient()),
		)
		if err != nil {
			c.GetLogger().Fatal().Err(err).Msg("failed to create generated http client")
		}

		c.generatedHTTPInternClient = client
	}

	return c.generatedHTTPInternClient
}

// mockExternRoundTripper is a custom type that implements the http.RoundTripper interface.
// We can set the `fn` field to any function we want for different test cases.
type mockInternRoundTripper struct {
	fn func(req *http.Request) (*http.Response, error)
}

// RoundTrip executes the mock function.
func (m *mockInternRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.fn(req)
}

func (c *Container) GetMockHTTPInternClient() *http.Client {
	if c.httpInternClient == nil {
		c.httpInternClient = &http.Client{
			Transport: &mockInternRoundTripper{
				fn: func(req *http.Request) (*http.Response, error) {
					if req.Method == http.MethodHead {
						header := make(http.Header)
						header.Set("Content-Length", "1024")
						return &http.Response{
							StatusCode:    http.StatusOK,
							Body:          io.NopCloser(bytes.NewBufferString("")),
							Header:        header,
							ContentLength: 1024,
						}, nil
					}
					// Create and return a mock response for GET (download)
					mockData := make([]byte, 1024)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(mockData)),
					}, nil
				},
			},
		}
	}

	return c.httpInternClient
}
