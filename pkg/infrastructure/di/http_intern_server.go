// nolint: dupl // normal to have the internal and external servers very similar
package di

import (
	"net/http"

	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
)

func (c *Container) GetHTTPInternServer() *http.Server {
	if c.httpInternServer == nil {
		swagger, err := intern.GetSpec()
		if err != nil {
			c.GetLogger().Fatal().Err(err).Msg("failed to get swagger")
		}

		mainRouter := http.NewServeMux()

		// Healthcheck endpoint for liveness status
		mainRouter.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})

		apiRouter := http.NewServeMux()

		// Register the intern handlers
		intern.HandlerFromMux(intern.NewStrictHandler(c.getHTTPInternResolver(), nil), apiRouter)

		validatorOptions := &middleware.Options{
			ErrorHandler: func(writer http.ResponseWriter, message string, statusCode int) {
				c.GetLogger().Error().
					Str("message", message).
					Int("status_code", statusCode).
					Msg("OAPI request validation error")

				http.Error(writer, message, statusCode)
			},
			DoNotValidateServers: true,
		}

		// Use the middleware to check all requests
		mainRouter.Handle(
			c.GetRootInternAPIPath()+"/",
			http.StripPrefix(
				c.GetRootInternAPIPath(),
				middleware.OapiRequestValidatorWithOptions(swagger, validatorOptions)(apiRouter),
			),
		)

		c.httpInternServer = &http.Server{
			Handler:   mainRouter,
			Addr:      c.config.Intern.Addr,
			TLSConfig: c.getInternTLSConfig(),
		}
	}

	return c.httpInternServer
}
