//nolint:lll,dupl // normal to have handlers very similar
package handler

import (
	"context"
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
		return h.genDownloadSolutionArchiveResponseObjectFromError(errors.Stamp(err))
	}

	solutionArchiveFile, err := h.uc.Execute(&solutionArchivePart)
	if err != nil {
		return h.genDownloadSolutionArchiveResponseObjectFromError(errors.Stamp(err))
	}

	return intern.DownloadSolutionArchive200ApplicationoctetStreamResponse{
		Body:          solutionArchiveFile.File,
		ContentLength: solutionArchiveFile.Size,
	}, nil
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

	switch int(apiErr.Identifier / 1000) {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		// TODO: Fix the status code on https://scality.atlassian.net/browse/ARTESCA-13615
		// The status code should be 422, but the generated code uses 400.
		// However, change this will demand changes the OpenAPI Spec.
		return intern.DownloadSolutionArchive400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: intern.BadRequestApplicationProblemPlusJSONResponse(
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
		return nil, errors.Stamp(err)
	}
}
