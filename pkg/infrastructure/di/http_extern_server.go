// nolint: dupl // normal to have the internal and external servers very similar
package di

import (
	"net/http"
	"strings"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/handler"
)

func (c *Container) GetHTTPExternServer() *http.Server {
	if c.httpExternServer == nil {
		mainRouter := http.NewServeMux()

		// Healthcheck endpoint for liveness status
		mainRouter.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})

		apiRouter := http.NewServeMux()

		// Add a middleware to check if the SolutionArchive version parameter
		// is present in the path
		// SolutionArchive name cannot be validated because the router send http/301
		// when the sequence "//" is sent in the path before any middleware execution
		validateVersion := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				urlParts := strings.Split(request.URL.Path, "/")
				if len(urlParts) < 4 {
					handler.WriteExternProblemDetails(
						request.Context(),
						c.GetLogger(),
						writer,
						errors.Wrap(
							domain.ErrHandlerBadRequest,
							errors.WithIdentifier(http.StatusBadRequest),
							errors.WithDetail("path is not valid"),
						),
						"path is not valid",
					)
					return
				}

				// Verify SolutionArchive version is present
				version := urlParts[3]
				if version == "" {
					handler.WriteExternProblemDetails(
						request.Context(),
						c.GetLogger(),
						writer,
						errors.Wrap(
							domain.ErrHandlerBadRequest,
							errors.WithIdentifier(http.StatusBadRequest),
							errors.WithDetail("Version parameter is required"),
						),
						"Version parameter is required",
					)
					return
				}

				next.ServeHTTP(writer, request)
			})
		}

		// Add a middleware to check if the body size (through Content-Length header)
		// 	is conform to the chunk size declared by the Content-Range header
		validateBodySizeAgainstContentRange := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				// OpenAPI-codegen validation will catch the missing Content-Range header
				contentRange := request.Header.Get("Content-Range")
				headerStart, headerEnd, _, err := library.ParseContentRange(contentRange)
				if err != nil {
					handler.WriteExternProblemDetails(
						request.Context(),
						c.GetLogger(),
						writer,
						errors.Wrap(err, errors.WithIdentifier(http.StatusBadRequest)),
						"Invalid Content-Range header")
					return
				}
				rangeSize := headerEnd - headerStart + 1
				// golang net/http library deals with missing or malformed Content-Length headers
				// by setting request.ContentLength to 0 or -1 so the comparison below is sufficient.
				if rangeSize != request.ContentLength {
					bodySizeErr := errors.Wrap(domain.ErrHandlerBadRequest,
						errors.WithIdentifier(http.StatusBadRequest),
						errors.WithDetail("Content-Range header does not match the body size"),
						errors.WithProperty("content_range_size", rangeSize),
						errors.WithProperty("content_length", request.ContentLength),
					)
					handler.WriteExternProblemDetails(
						request.Context(),
						c.GetLogger(),
						writer,
						bodySizeErr,
						"Body size does not match Content-Range",
					)
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

		// Use a custom middleware to check all requests
		// OpenAPI-codegen middleware validator cannot be used for the moment
		// because it reads the full body through io.ReadAll().
		// In the version 1.1.2 of the library, the option "Skipper" is not yet available.
		// This middleware must be executed as soon as possible to avoid the router to return a http/404
		// because the path has not been found.
		mainRouter.Handle(
			c.getRootExternAPIPath()+"/",
			http.StripPrefix(
				c.getRootExternAPIPath(),
				validateVersion(apiRouter),
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
