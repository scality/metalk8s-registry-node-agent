package handler

import (
	"fmt"
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

// fillUploadChunkSuccessResponseFromArtifactStatus fills the UploadChunkSuccessResponse object
// from the domain.ArtifactStatus object.
func fillUploadChunkSuccessResponseFromArtifactStatus(
	dst *extern.UploadChunkSuccessResponse,
	src *domain.ArtifactStatus,
) {
	size := int(src.Artifact.Size)

	var uploadChuncks []extern.UploadChunkRange

	fillUploadChunkRangesFromDomainPartMetas(&uploadChuncks, src.Parts)

	isCompleted := src.IsComplete()

	dst.Artifact = &src.Artifact.Name
	dst.Version = &src.Artifact.Version
	dst.Size = &size
	dst.UploadedChunks = &uploadChuncks
	dst.Sha256sum = &src.Artifact.Hash
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
			WithDetail("Body is missing.").
			Throw()
	}

	if src.Artifact == "" {
		return errors.From(domain.ErrHandlerMissingRequestParameter).
			WithIdentifier(400003).
			WithDetail("Parameter 'artifact' is missing.").
			Throw()
	}

	if src.Params.XSha256Checksum == "" {
		return errors.From(domain.ErrHandlerMissingRequestHeader).
			WithIdentifier(400002).
			WithDetail("Header 'X-Sha256-checksum' is missing.").
			Throw()
	}

	if src.Params.XTargetVersion == "" {
		return errors.From(domain.ErrHandlerMissingRequestHeader).
			WithIdentifier(400002).
			WithDetail("Header 'X-Target-Version' is missing.").
			Throw()
	}

	start, end, total, err := parseContentRange(src.Params.ContentRange)
	if err != nil {
		return errors.Stamp(err)
	}

	dst.Artifact = &domain.Artifact{
		Name:    src.Artifact,
		Version: src.Params.XTargetVersion,
		Size:    total,
		Hash:    src.Params.XSha256Checksum,
	}

	dst.Meta = &domain.PartMeta{
		Start: start,
		End:   end,
	}

	dst.Content = src.Body

	return nil
}

// fillProblemDetailsFromAPIErrorsError fills the ProblemDetails object from the apierrors.Error object.
func (h *UploadPart) fillProblemDetailsFromAPIErrorsError(
	dst *extern.ProblemDetails,
	src *errors.Error,
) {
	h.logger.Error().Err(src).Msg("Uploads API error")

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
