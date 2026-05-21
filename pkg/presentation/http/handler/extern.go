// nolint: dupl // normal to have the internal and external handlers very similar
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
)

// fillUploadChunkRangesFromDomainPartMetas fills the UploadChunkRange slice from the domain.Part map.
func fillUploadChunkRangesFromDomainPartMetas(
	dst *[]extern.UploadChunkRange,
	src map[int64]*domain.PartMeta,
) {
	*dst = make([]extern.UploadChunkRange, len(src))

	i := 0
	for _, part := range src {
		fillUploadChunkRangeFromDomainPartMeta(&(*dst)[i], part)
		i++
	}
}

// fillUploadChunkRangeFromDomainPartMeta fills the UploadChunkRange object from the domain.Part object.
func fillUploadChunkRangeFromDomainPartMeta(
	dst *extern.UploadChunkRange,
	src *domain.PartMeta,
) {
	start := int(src.Start)
	end := int(src.End)
	size := int(src.Size())
	dst.RangeStartIndex = &start
	dst.RangeSize = &size
	dst.RangeEndIndex = &end
}

// fillUploadChunkSuccessResponseFromSolutionArchiveStatus fills the UploadChunkSuccessResponse object
// from the domain.SolutionArchiveStatus object.
func fillUploadChunkSuccessResponseFromSolutionArchiveStatus(
	dst *extern.UploadChunkSuccessResponse,
	src *domain.SolutionArchiveStatus,
) {
	size := int(src.SolutionArchive.Size)

	var uploadChuncks []extern.UploadChunkRange

	fillUploadChunkRangesFromDomainPartMetas(&uploadChuncks, src.Parts)

	isCompleted := src.IsComplete()

	dst.SolutionArchive = &src.SolutionArchive.Name
	dst.Version = &src.SolutionArchive.Version
	dst.Size = &size
	dst.UploadedChunks = &uploadChuncks
	if src.SolutionArchive.Hash != nil {
		dst.Sha256sum = src.SolutionArchive.Hash
	}
	dst.IsCompleted = &isCompleted
}

// fillPartFromUploadChunkRequestObject fills the Part object from the UploadChunkRequestObject object.
func fillPartFromUploadChunkRequestObject(
	dst *domain.Part,
	src *extern.UploadChunkRequestObject,
) error {
	if src.Body == nil {
		return errors.From(domain.ErrHandlerBadRequest).
			WithIdentifier(400000).
			WithDetail("body is missing").
			Throw()
	}

	start, end, total, err := ParseContentRange(src.Params.ContentRange)
	if err != nil {
		return errors.Stamp(err)
	}

	dst.SolutionArchive = &domain.SolutionArchive{
		Name:    src.SolutionArchive,
		Version: src.Version,
		Size:    total,
	}

	dst.Meta = &domain.PartMeta{
		Start: start,
		End:   end,
	}

	dst.Content = src.Body

	return nil
}

// fillExternProblemDetailsFromAPIErrorsError fills the ProblemDetails object from the apierrors.Error object.
func fillExternProblemDetailsFromAPIErrorsError(
	ctx context.Context,
	logger *slog.Logger,
	dst *extern.ProblemDetails,
	src *errors.Error,
	logMsg string,
) {
	logger.ErrorContext(ctx, logMsg, slog.Any("error_message", src))

	status := src.Identifier / 1000 // nolint: gosec // TODO: Refactor this in the "polishing" sprint.
	code := fmt.Sprintf("%d", src.Identifier)

	dst.Title = src.Title
	dst.Status = &status
	dst.Code = &code

	if len(src.Details) > 0 {
		detail := strings.Join(src.Details, ": ")
		dst.Detail = &detail
	}

	if instance, ok := src.Properties["instance"]; ok {
		instanceStr, ok := instance.(string)
		if ok {
			dst.Instance = &(instanceStr)
		}
	}

	if len(src.Properties) > 0 {
		fillInstanceFromPropertiesMap(&dst.Instance, src.Properties)

		errs := make(extern.Errors, 0, len(src.Properties))
		fillExternErrorDetailsFromPropertiesMap(&errs, src.Properties)

		dst.Errors = &errs
	}
}

func (h *UploadPart) fillProblemDetailsFromAPIErrorsError(
	ctx context.Context,
	dst *extern.ProblemDetails,
	src *errors.Error,
) {
	fillExternProblemDetailsFromAPIErrorsError(ctx, h.logger, dst, src, "Uploads API error")
}

// WriteExternProblemDetails writes a ProblemDetails JSON response to w based
// on the given error. The HTTP status code is derived from the error
// identifier (status = identifier / 1000). If err does not wrap an
// *errors.Error, a generic 500 ProblemDetails is returned instead.
func WriteExternProblemDetails(
	ctx context.Context,
	logger *slog.Logger,
	w http.ResponseWriter,
	err error,
	logMsg string,
) {
	var apiErr *errors.Error
	if !errors.As(err, &apiErr) {
		apiErr = errors.From(domain.ErrUnknown).
			WithIdentifier(500000).
			WithDetail(err.Error()).
			Throw().(*errors.Error)
	}

	var problemDetails extern.ProblemDetails

	fillExternProblemDetailsFromAPIErrorsError(ctx, logger, &problemDetails, apiErr, logMsg)

	var status int
	if problemDetails.Status != nil {
		status = int(*problemDetails.Status)
	}
	if status < 100 || status > 599 {
		status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	if encErr := json.NewEncoder(w).Encode(problemDetails); encErr != nil {
		logger.ErrorContext(ctx, "failed to encode ProblemDetails response", slog.Any("error_message", encErr))
	}
}

// fillExternErrorDetailsFromPropertiesMap fills the ErrorDetail slice from a map[string]any.
func fillExternErrorDetailsFromPropertiesMap(
	dst *extern.Errors,
	src map[string]any,
) {
	*dst = make([]extern.ErrorDetail, 0, len(src))

	for key, value := range src {
		if strings.HasPrefix(key, "problem_") {
			var errorDetail extern.ErrorDetail

			fillExternErrorDetailFromProperty(&errorDetail, key, value)

			*dst = append(*dst, errorDetail)
		}
	}
}

// fillExternErrorDetailFromProperty fills the ErrorDetail object from a property name and value.
func fillExternErrorDetailFromProperty(
	dst *extern.ErrorDetail,
	key string, value any,
) {
	code := key
	detail := fmt.Sprintf("%v", value)

	dst.Code = &code
	dst.Detail = detail
}
