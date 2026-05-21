// nolint: dupl // normal to have the internal and external servers very similar
package di

import (
	"log/slog"
	"net/http"
	"os"

	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
)

func (c *Container) GetHTTPInternServer() *http.Server {
	if c.httpInternServer == nil {
		swagger, err := intern.GetSwagger()
		if err != nil {
			c.GetLogger().ErrorContext(c.ctx, "failed to get swagger", slog.Any("error_message", err))
			os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
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
				c.GetLogger().ErrorContext(c.ctx, "OAPI request validation error",
					slog.String("message", message),
					slog.Int("status_code", statusCode),
				)

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
