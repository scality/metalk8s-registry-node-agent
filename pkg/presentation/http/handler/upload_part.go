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
		return h.genUploadChunkResponseObjectFromError(
			errors.Wrap(err,
				errors.WithIdentifier(http.StatusBadRequest),
			),
		)
	}

	solutionArchiveStatus, err := h.uc.Execute(&part)
	if err != nil {
		// Default value for the API error
		apiErr := errors.Wrap(err,
			errors.WithIdentifier(http.StatusInternalServerError),
		)
		if errors.IdentifierStartsWith(err, "232-122") {
			apiErr = errors.Wrap(err, errors.WithIdentifier(http.StatusNotFound))
		} else if errors.IdentifierStartsWith(err, "233-27") {
			apiErr = errors.Wrap(err, errors.WithIdentifier(http.StatusNotFound))
		} else if errors.IdentifierStartsWith(err, "235") {
			apiErr = errors.Wrap(err, errors.WithIdentifier(http.StatusNotFound))
		} else if errors.IdentifierStartsWith(err, "238-192-181") {
			apiErr = errors.Wrap(err, errors.WithIdentifier(http.StatusUnprocessableEntity))
		} else if errors.IdentifierStartsWith(err, "238-192-183") {
			apiErr = errors.Wrap(err, errors.WithIdentifier(http.StatusUnprocessableEntity))
		}

		return h.genUploadChunkResponseObjectFromError(apiErr)
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

	// no need to test for bad conversion
	// the error is an underlying *errors.Error, wrapped in UploadChunk
	errors.As(err, &apiErr) // nolint: errcheck

	h.fillProblemDetailsFromAPIErrorsError(&problemDetails, apiErr)

	switch int(*problemDetails.Status) {
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
