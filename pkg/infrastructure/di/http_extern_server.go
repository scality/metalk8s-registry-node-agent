// nolint: dupl // normal to have the internal and external servers very similar
package di

import (
	"context"
	"net/http"

	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/handler"
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

		// Add a middleware to check if the body size (through Content-Length header)
		// 	is conform to the chunk size declared by the Content-Range header
		validateBodySizeAgainstContentRange := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				// OpenAPI-codegen validation will catch the missing Content-Range header
				contentRange := request.Header.Get("Content-Range")
				headerStart, headerEnd, _, err := handler.ParseContentRange(contentRange)
				if err != nil {
					http.Error(writer, "Invalid Content-Range", http.StatusBadRequest)
					return
				}
				rangeSize := headerEnd - headerStart + 1
				// golang net/http library deals with missing or malformed Content-Length headers
				// by setting request.ContentLength to 0 or -1 so the comparison below is sufficient.
				if rangeSize != request.ContentLength {
					http.Error(writer, "Content-Range header does not match the body size", http.StatusBadRequest)
					return
				}

				next.ServeHTTP(writer, request)
			})
		}

		// OpenAPI-codegen generate.go returns an InternalServerError error when middleware
		// from strictHandler fails.
		// As we want a Bad Request error in this case, we need to define the middleware
		// in the ServerInterfaceWrapper.
		middlewares := []extern.MiddlewareFunc{
			validateBodySizeAgainstContentRange,
		}

		// Register the extern handlers
		extern.HandlerWithOptions(
			extern.NewStrictHandler(c.getHTTPResolver(), nil),
			extern.StdHTTPServerOptions{
				BaseRouter:  apiRouter,
				Middlewares: middlewares,
			},
		)

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
