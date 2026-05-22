// nolint: dupl // normal to have the internal and external handlers very similar
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
	"k8s.io/utils/ptr"
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
		return errors.Wrap(domain.ErrHandlerBadRequest,
			errors.WithIdentifier(400000),
			errors.WithDetail("body is missing"),
		)
	}

	start, end, total, err := ParseContentRange(src.Params.ContentRange)
	if err != nil {
		return errors.Wrap(err)
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
	logger *zerolog.Logger,
	dst *extern.ProblemDetails,
	src *errors.Error,
	logMsg string,
) {
	logger.Error().Err(src).Msg(logMsg)

	// The HTTP status is the last error identifier.
	status := int32(src.Identifier[len(src.Identifier)-1])
	code := src.GetIdentifier()

	dst.Title = src.Title
	dst.Status = &status
	dst.Code = &code

	details := []string{}
	details = append(details, src.Details...)
	dst.Detail = ptr.To(strings.Join(details, ": "))

	if instance, ok := src.Properties["instance"]; ok {
		instanceStr, ok := instance.(string)
		if ok {
			dst.Instance = &(instanceStr)
		}
	}

	for key, value := range src.Properties {
		// We consider that properties containing "path" in their name are sensitive information.
		if !strings.Contains(key, "path") && key != "instance" {
			if dst.AdditionalInformation == nil {
				dst.AdditionalInformation = &map[string]interface{}{}
			}
			(*dst.AdditionalInformation)[key] = value
		}
	}
}

func (h *UploadPart) fillProblemDetailsFromAPIErrorsError(
	dst *extern.ProblemDetails,
	src *errors.Error,
) {
	fillExternProblemDetailsFromAPIErrorsError(h.logger, dst, src, "Uploads API error")
}

// WriteExternProblemDetails writes a ProblemDetails JSON response to w based
// on the given error. The HTTP status code is derived from the error
// identifier (status = identifier / 1000). If err does not wrap an
// *errors.Error, a generic 500 ProblemDetails is returned instead.
func WriteExternProblemDetails(
	logger *zerolog.Logger,
	w http.ResponseWriter,
	err error,
	logMsg string,
) {
	var apiErr *errors.Error
	if !errors.As(err, &apiErr) {
		apiErr = errors.Wrap(domain.ErrUnknown,
			errors.WithIdentifier(http.StatusInternalServerError),
			errors.WithDetail(err.Error()),
		).(*errors.Error)
	}

	var problemDetails extern.ProblemDetails

	fillExternProblemDetailsFromAPIErrorsError(logger, &problemDetails, apiErr, logMsg)

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
		logger.Error().Err(encErr).Msg("failed to encode ProblemDetails response")
	}
}
