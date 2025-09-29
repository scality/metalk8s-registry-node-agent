package di

import (
	"net/http"

	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/generated"
)

func (c *Container) GetHTTPServer() *http.Server {
	if c.httpServer == nil {
		swagger, err := generated.GetSwagger()
		if err != nil {
			c.logger.Fatal().Err(err).Msg("failed to get swagger")
		}

		mainRouter := http.NewServeMux()

		// Healthcheck endpoint for liveness status
		mainRouter.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})

		apiRouter := http.NewServeMux()

		// Register the generated handlers
		generated.HandlerFromMux(generated.NewStrictHandler(c.getHTTPResolver(), nil), apiRouter)

		validatorOptions := &middleware.Options{
			ErrorHandler: func(writer http.ResponseWriter, message string, statusCode int) {
				c.logger.Error().
					Str("message", message).
					Int("status_code", statusCode).
					Msg("OAPI request validation error")

				http.Error(writer, message, statusCode)
			},
			DoNotValidateServers: true,
		}

		// Use the middleware to check all requests
		mainRouter.Handle(
			c.GetRootAPIPath()+"/",
			http.StripPrefix(
				c.GetRootAPIPath(),
				middleware.OapiRequestValidatorWithOptions(swagger, validatorOptions)(apiRouter),
			),
		)

		c.httpServer = &http.Server{
			Handler: mainRouter,
			Addr:    c.config.HTTP.Addr,
		}
	}

	return c.httpServer
}
