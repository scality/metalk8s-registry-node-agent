//nolint:lll
package handler

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/usecase"
)

type UploadPart struct {
	logger *zerolog.Logger

	uc *usecase.UploadPart
}

func NewUploadPart(
	logger *zerolog.Logger,
	uc *usecase.UploadPart,
) *UploadPart {
	l := logger.With().Str("http_handler", "upload_part").Logger()

	return &UploadPart{
		logger: &l,
		uc:     uc,
	}
}

//nolint:ireturn // Generated code forces to return an interface.
func (h *UploadPart) UploadChunk(
	_ context.Context,
	request extern.UploadChunkRequestObject,
) (extern.UploadChunkResponseObject, error) {
	var part domain.Part

	if err := fillPartFromUploadChunkRequestObject(&part, &request); err != nil {
		return h.genUploadChunkResponseObjectFromError(errors.Wrap(err))
	}

	solutionArchiveStatus, err := h.uc.Execute(&part)
	if err != nil {
		return h.genUploadChunkResponseObjectFromError(errors.Wrap(err))
	}

	var response extern.UploadChunkSuccessResponse

	fillUploadChunkSuccessResponseFromSolutionArchiveStatus(&response, solutionArchiveStatus)

	return extern.UploadChunk200JSONResponse(response), nil
}

// genUploadChunkResponseObjectFromError generates a proper AbortUploadSessionResponseObject
// from a given error.
//
//nolint:funlen,ireturn // Needs refactoring
func (h *UploadPart) genUploadChunkResponseObjectFromError(
	err error,
) (extern.UploadChunkResponseObject, error) {
	var apiErr *errors.Error

	var problemDetails extern.ProblemDetails

	errors.As(err, &apiErr)

	h.fillProblemDetailsFromAPIErrorsError(&problemDetails, apiErr)

	switch int(apiErr.Identifier / 1000) {
	case http.StatusBadRequest:
		return extern.UploadChunk400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: extern.BadRequestApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusUnprocessableEntity:
		return extern.UploadChunk422ApplicationProblemPlusJSONResponse{
			UnprocessableEntityApplicationProblemPlusJSONResponse: extern.UnprocessableEntityApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusUnauthorized:
		return extern.UploadChunk401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: extern.UnauthorizedApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusForbidden:
		return extern.UploadChunk403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: extern.ForbiddenApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusNotFound:
		return extern.UploadChunk404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: extern.NotFoundApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusInternalServerError:
		return extern.UploadChunk500ApplicationProblemPlusJSONResponse{
			ServerErrorApplicationProblemPlusJSONResponse: extern.ServerErrorApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusInsufficientStorage:
		return extern.UploadChunk507ApplicationProblemPlusJSONResponse{
			InsufficientStorageApplicationProblemPlusJSONResponse: extern.InsufficientStorageApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	default:
		return nil, errors.Wrap(err)
	}
}
