// nolint: dupl // normal to have the internal and external servers very similar
package di

import (
	"context"
	"net/http"

	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
)

func (c *Container) GetHTTPExternServer() *http.Server {
	if c.httpExternServer == nil {
		swagger, err := extern.GetSwagger()
		if err != nil {
			c.GetLogger().Fatal().Err(err).Msg("failed to get swagger")
		}

		mainRouter := http.NewServeMux()

		// Healthcheck endpoint for liveness status
		mainRouter.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})

		apiRouter := http.NewServeMux()

		// Register the extern handlers
		extern.HandlerFromMux(extern.NewStrictHandler(c.getHTTPResolver(), nil), apiRouter)

		validatorOptions := &middleware.Options{
			ErrorHandlerWithOpts: func(
				ctx context.Context,
				err error,
				w http.ResponseWriter,
				r *http.Request,
				opts middleware.ErrorHandlerOpts,
			) {
				statusCode := opts.StatusCode
				if opts.MatchedRoute == nil {
					// request URL is probably missing solutionArchive and/or version parameter(s)
					statusCode = http.StatusBadRequest
				}
				c.GetLogger().Error().
					Str("message", err.Error()).
					Int("status_code", statusCode).
					Msg("OAPI request validation error")

				http.Error(w, err.Error(), statusCode)
			},
			DoNotValidateServers: true,
		}

		// Use the middleware to check all requests
		mainRouter.Handle(
			c.GetRootExternAPIPath()+"/",
			http.StripPrefix(
				c.GetRootExternAPIPath(),
				middleware.OapiRequestValidatorWithOptions(swagger, validatorOptions)(apiRouter),
			),
		)

		c.httpExternServer = &http.Server{
			Handler:   mainRouter,
			Addr:      c.config.Extern.Addr,
			TLSConfig: c.getExternTLSConfig(),
		}
	}

	return c.httpExternServer
}
