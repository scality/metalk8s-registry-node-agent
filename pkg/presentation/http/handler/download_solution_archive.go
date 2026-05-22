//nolint:lll,dupl // normal to have handlers very similar
package handler

import (
	"context"
	"crypto"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/usecase"
)

type DownloadSolutionArchive struct {
	logger *zerolog.Logger

	uc *usecase.DownloadSolutionArchive
}

func NewDownloadSolutionArchive(
	logger *zerolog.Logger,
	uc *usecase.DownloadSolutionArchive,
) *DownloadSolutionArchive {
	l := logger.With().Str("http_handler", "download_solution_archive").Logger()

	return &DownloadSolutionArchive{
		logger: &l,
		uc:     uc,
	}
}

//nolint:ireturn // Generated code forces to return an interface.
func (h *DownloadSolutionArchive) DownloadSolutionArchive(
	_ context.Context,
	request intern.DownloadSolutionArchiveRequestObject,
) (intern.DownloadSolutionArchiveResponseObject, error) {
	var solutionArchivePart domain.Part

	if err := fillSolutionArchiveFromDownloadSolutionArchiveRequestObject(&solutionArchivePart, &request); err != nil {
		return h.genDownloadSolutionArchiveResponseObjectFromError(
			errors.Wrap(err,
				errors.WithIdentifier(http.StatusBadRequest),
			),
		)
	}

	partFile, err := h.uc.Execute(&solutionArchivePart)
	if err != nil {
		// Default value for the API error
		apiErr := errors.Wrap(err,
			errors.WithIdentifier(http.StatusInternalServerError),
		)
		if errors.Is(err, errors.Wrap(domain.ErrNotFound, errors.WithIdentifier(404000))) {
			apiErr = errors.Wrap(err, errors.WithIdentifier(http.StatusNotFound))
		}

		return h.genDownloadSolutionArchiveResponseObjectFromError(apiErr)

	}

	// As oapi-codegen does not handle HTTP Trailers, we need to manually implement this point,
	// by using a specific struct and implement the VisitDownloadSolutionArchiveResponse method.
	return &downloadWithTrailer{
		file:         partFile.File,
		contentRange: fmt.Sprintf("bytes %d-%d/%d", solutionArchivePart.Meta.Start, solutionArchivePart.Meta.End, partFile.Size),
	}, nil
}

// downloadWithTrailer streams the response body while computing the SHA-256
// digest on-the-fly via io.TeeReader, then sends Content-Digest as an HTTP trailer.
//
// Note: Content-Length is intentionally not set on the response. HTTP/1.x trailers
// require chunked transfer encoding, which Go's net/http server only uses when
// Content-Length is absent. The total size is still communicated via Content-Range.
type downloadWithTrailer struct {
	file         io.ReadCloser
	contentRange string
}

func (r *downloadWithTrailer) VisitDownloadSolutionArchiveResponse(w http.ResponseWriter) error {
	defer r.file.Close() //nolint:errcheck // Best-effort; response is already being written.

	hasher := crypto.SHA256.New()
	tee := io.TeeReader(r.file, hasher)

	w.Header().Set("Trailer", "Content-Digest")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Range", r.contentRange)
	w.WriteHeader(http.StatusPartialContent)

	if _, err := io.Copy(w, tee); err != nil {
		return err
	}

	w.Header().Set("Content-Digest", fmt.Sprintf("sha-256=:%s:", base64.StdEncoding.EncodeToString(hasher.Sum(nil))))

	return nil
}

// genDownloadSolutionArchiveResponseObjectFromError generates a proper
// DownloadSolutionArchiveResponseObject from a given error.
//
//nolint:funlen,ireturn // Needs refactoring
func (h *DownloadSolutionArchive) genDownloadSolutionArchiveResponseObjectFromError(
	err error,
) (intern.DownloadSolutionArchiveResponseObject, error) {
	var apiErr *errors.Error

	var problemDetails intern.ProblemDetails

	errors.As(err, &apiErr)

	h.fillProblemDetailsFromAPIErrorsError(&problemDetails, apiErr)

	switch int(*problemDetails.Status) {
	case http.StatusBadRequest:
		return intern.DownloadSolutionArchive400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: intern.BadRequestApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusUnprocessableEntity:
		return intern.DownloadSolutionArchive422ApplicationProblemPlusJSONResponse{
			UnprocessableEntityApplicationProblemPlusJSONResponse: intern.UnprocessableEntityApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusUnauthorized:
		return intern.DownloadSolutionArchive401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: intern.UnauthorizedApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusForbidden:
		return intern.DownloadSolutionArchive403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: intern.ForbiddenApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusNotFound:
		return intern.DownloadSolutionArchive404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: intern.NotFoundApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusInternalServerError:
		return intern.DownloadSolutionArchive500ApplicationProblemPlusJSONResponse{
			ServerErrorApplicationProblemPlusJSONResponse: intern.ServerErrorApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	default:
		return nil, errors.Wrap(err)
	}
}
