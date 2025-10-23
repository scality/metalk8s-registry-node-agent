//nolint:lll
package handler

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/generated"
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
	request generated.UploadChunkRequestObject,
) (generated.UploadChunkResponseObject, error) {
	var part domain.Part

	if err := fillPartFromUploadChunkRequestObject(&part, &request); err != nil {
		return h.genUploadChunkResponseObjectFromError(domain.Stamp(err))
	}

	artifactStatus, err := h.uc.Execute(&part)
	if err != nil {
		return h.genUploadChunkResponseObjectFromError(domain.Stamp(err))
	}

	var response generated.UploadChunkSuccessResponse

	fillUploadChunkSuccessResponseFromArtifactStatus(&response, artifactStatus)

	return generated.UploadChunk200JSONResponse(response), nil
}

// genUploadChunkResponseObjectFromError generates a proper AbortUploadSessionResponseObject
// from a given error.
//
//nolint:funlen,ireturn // Needs refactoring
func (h *UploadPart) genUploadChunkResponseObjectFromError(
	err error,
) (generated.UploadChunkResponseObject, error) {
	var apiErr *domain.Error

	var problemDetails generated.ProblemDetails

	errors.As(err, &apiErr)

	h.fillProblemDetailsFromAPIErrorsError(&problemDetails, apiErr)

	switch apiErr.Status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		// TODO: Fix the status code on https://scality.atlassian.net/browse/ARTESCA-13615
		// The status code should be 422, but the generated code uses 400.
		// However, change this will demand changes the OpenAPI Spec.
		return generated.UploadChunk400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: generated.BadRequestApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusUnauthorized:
		return generated.UploadChunk401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: generated.UnauthorizedApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusForbidden:
		return generated.UploadChunk403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: generated.ForbiddenApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusNotFound:
		return generated.UploadChunk404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: generated.NotFoundApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusInternalServerError:
		return generated.UploadChunk500ApplicationProblemPlusJSONResponse{
			ServerErrorApplicationProblemPlusJSONResponse: generated.ServerErrorApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusInsufficientStorage:
		return generated.UploadChunk507ApplicationProblemPlusJSONResponse{
			InsufficientStorageApplicationProblemPlusJSONResponse: generated.InsufficientStorageApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	default:
		return nil, domain.Stamp(err)
	}
}
