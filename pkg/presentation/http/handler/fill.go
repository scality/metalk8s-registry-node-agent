//nolint:dupl,wrapcheck,funlen,lll,revive // Duplication is fine, too much wraps missing
package handler

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/generated"
)

// fillUploadChunkRangesFromDomainPartMetas fills the UploadChunkRange slice from the domain.Part map.
func fillUploadChunkRangesFromDomainPartMetas(
	dst *[]generated.UploadChunkRange,
	src map[int64]*domain.PartMeta,
) {
	*dst = make([]generated.UploadChunkRange, len(src))

	i := 0
	for _, part := range src {
		fillUploadChunkRangeFromDomainPartMeta(&(*dst)[i], part)
		i++
	}
}

// fillUploadChunkRangeFromDomainPartMeta fills the UploadChunkRange object from the domain.Part object.
func fillUploadChunkRangeFromDomainPartMeta(
	dst *generated.UploadChunkRange,
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
	dst *generated.UploadChunkSuccessResponse,
	src *domain.ArtifactStatus,
) {
	size := int(src.Artifact.Size)

	var uploadChuncks []generated.UploadChunkRange

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
	src *generated.UploadChunkRequestObject,
) error {
	if src.Body == nil {
		return domain.FromTemplate(domain.ErrHandlerBadRequestError).
			WithDetail("Body is missing.").
			Throw()
	}

	if src.Artifact == "" {
		return domain.FromTemplate(domain.ErrHandlerMissingRequestParameterError).
			WithDetail("Parameter 'artifact' is missing.").
			Throw()
	}

	if src.Params.XSha256Checksum == "" {
		return domain.FromTemplate(domain.ErrHandlerMissingRequestHeaderError).
			WithDetail("Header 'X-Sha256-checksum' is missing.").
			Throw()
	}

	if src.Params.XTargetVersion == "" {
		return domain.FromTemplate(domain.ErrHandlerMissingRequestHeaderError).
			WithDetail("Header 'X-Target-Version' is missing.").
			Throw()
	}

	start, end, total, err := parseContentRange(src.Params.ContentRange)
	if err != nil {
		return domain.Stamp(err)
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

// contentRangeRegexp is a regular expression to parse the Content-Range header.
var contentRangeRegexp = regexp.MustCompile(
	`bytes (?P<begin>[0-9]{1,12})-(?P<end>[0-9]{1,12})\/(?P<total>[0-9]{1,12})`,
)

const expectedContentRangeSubmatches = 4

// parseContentRange() parses the UploadChunkRange object and returns the start,
// size and end values.
// nolint:revive
func parseContentRange(
	contentRange string,
) (int64, int64, int64, error) {
	var start, end, total int64
	var err error

	// Check if the Content-Range header is missing
	if contentRange == "" {
		err = domain.FromTemplate(domain.ErrHandlerMissingRequestHeaderError).
			WithDetail("Content-Range header is missing.").
			Throw()

		return start, end, total, err
	}

	matched := contentRangeRegexp.FindStringSubmatch(contentRange)

	if len(matched) != expectedContentRangeSubmatches {
		err = domain.FromTemplate(domain.ErrHandlerInvalidRequestHeaderFormatError).
			WithDetail("Content-Range header is not in the expected format.").
			AddProperty("received_content_range", contentRange).
			AddProperty("expected_content_range_format", "bytes <start>-<end>/<total>").
			AddProperty("example_content_range", "bytes 0-19/20").
			AddProperty("validation_regex", contentRangeRegexp.String()).
			Throw()

		return start, end, total, err
	}

	problems := make(map[string]any)

	start, err = strconv.ParseInt(matched[1], 10, 64)
	if err != nil {
		problems["problem_invalid_start"] = err.Error()
	}

	end, err = strconv.ParseInt(matched[2], 10, 64)
	if err != nil {
		problems["problem_invalid_end"] = err.Error()
	}

	total, err = strconv.ParseInt(matched[3], 10, 64)
	if err != nil {
		problems["problem_invalid_total"] = err.Error()
	}

	if start < 0 {
		problems["problem_invalid_start_less_than_zero"] = fmt.Sprintf("start < 0: %d", start)
	}

	if start > end {
		problems["problem_invalid_start_greater_than_end"] = fmt.Sprintf("start > end: %d > %d", start, end)
	}

	size := end - start + 1
	if size > total {
		problems["problem_invalid_size_greater_than_total"] = fmt.Sprintf("size > total: %d > %d", size, total)
	}

	if len(problems) > 0 {
		err = domain.FromTemplate(domain.ErrHandlerInvalidRequestHeaderFormatError).
			WithDetail("Content-Range header is not in the expected format.").
			WithProperties(problems).
			AddProperty("received_content_range", contentRange).
			AddProperty("expected_content_range_format", "bytes <start>-<end>/<total>").
			AddProperty("example_content_range", "bytes 0-19/20").
			AddProperty("validation_regex", contentRangeRegexp.String()).
			Throw()

		start, end, total = 0, 0, 0

		return start, end, total, err
	}

	return start, end, total, nil
}

// fillProblemDetailsFromAPIErrorsError fills the ProblemDetails object from the apierrors.Error object.
func (h *UploadPart) fillProblemDetailsFromAPIErrorsError(
	dst *generated.ProblemDetails,
	src *domain.Error,
) {
	h.logger.Error().Err(src).Msg("Uploads API error")

	status := int32(src.Status) // nolint: gosec // TODO: Refactor this in the "polishing" sprint.
	code := fmt.Sprintf("%d", src.Subclass)
	typeURL := src.Type

	dst.Title = src.Title
	dst.Status = &status
	dst.Code = &code
	dst.Type = &typeURL

	if src.Detail != "" {
		detail := src.Detail
		dst.Detail = &detail
	}

	if src.Instance != "" {
		instance := src.Instance
		dst.Instance = &instance
	}

	if len(src.Properties) > 0 {
		fillInstanceFromPropertiesMap(&dst.Instance, src.Properties)

		errs := make(generated.Errors, 0, len(src.Properties))
		fillErrorDetailsFromPropertiesMap(&errs, src.Properties)

		dst.Errors = &errs
	}
}

// fillInstanceFromPropertiesMap generates an instance string from a map[string]any.
func fillInstanceFromPropertiesMap(
	dst **string,
	src map[string]any,
) {
	b := bytes.NewBuffer(nil)

	if dst != nil {
		b.WriteString(**dst)
		b.WriteString(":")
	}

	for key, value := range src {
		if !strings.HasPrefix(key, "problem_") {
			fmt.Fprintf(b, "%s='%v',", key, value)
		}
	}

	if b.Len() > 0 {
		instance := b.String()[:b.Len()-1]
		*dst = &instance
	}
}

// fillErrorDetailsFromPropertiesMap fills the ErrorDetail alice from a map[string]any.
func fillErrorDetailsFromPropertiesMap(
	dst *generated.Errors,
	src map[string]any,
) {
	*dst = make([]generated.ErrorDetail, 0, len(src))

	for key, value := range src {
		if strings.HasPrefix(key, "problem_") {
			var errorDetail generated.ErrorDetail

			fillErrorDetailFromProperty(&errorDetail, key, value)

			*dst = append(*dst, errorDetail)
		}
	}
}

// fillErrorDetailFromProperty fills the ErrorDetail object from a property name and value.
func fillErrorDetailFromProperty(
	dst *generated.ErrorDetail,
	key string, value any,
) {
	code := key
	detail := fmt.Sprintf("%v", value)

	dst.Code = &code
	dst.Detail = detail
}
