//nolint:dupl,wrapcheck,funlen,lll,revive // Duplication is fine, too much wraps missing
package handler

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
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
		err = errors.From(domain.ErrHandlerMissingRequestHeader).
			WithIdentifier(400002).
			WithDetail("Content-Range header is missing.").
			Throw()

		return start, end, total, err
	}

	matched := contentRangeRegexp.FindStringSubmatch(contentRange)

	if len(matched) != expectedContentRangeSubmatches {
		err = errors.From(domain.ErrHandlerInvalidRequestHeaderFormat).
			WithIdentifier(400006).
			WithDetail("Content-Range header is not in the expected format.").
			WithProperty("received_content_range", contentRange).
			WithProperty("expected_content_range_format", "bytes <start>-<end>/<total>").
			WithProperty("example_content_range", "bytes 0-19/20").
			WithProperty("validation_regex", contentRangeRegexp.String()).
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
		err = errors.From(domain.ErrHandlerInvalidRequestHeaderFormat).
			WithIdentifier(400006).
			WithDetail("Content-Range header is not in the expected format.").
			WithProperties(problems).
			WithProperty("received_content_range", contentRange).
			WithProperty("expected_content_range_format", "bytes <start>-<end>/<total>").
			WithProperty("example_content_range", "bytes 0-19/20").
			WithProperty("validation_regex", contentRangeRegexp.String()).
			Throw()

		start, end, total = 0, 0, 0

		return start, end, total, err
	}

	return start, end, total, nil
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
	dst *extern.Errors,
	src map[string]any,
) {
	*dst = make([]extern.ErrorDetail, 0, len(src))

	for key, value := range src {
		if strings.HasPrefix(key, "problem_") {
			var errorDetail extern.ErrorDetail

			fillErrorDetailFromProperty(&errorDetail, key, value)

			*dst = append(*dst, errorDetail)
		}
	}
}

// fillErrorDetailFromProperty fills the ErrorDetail object from a property name and value.
func fillErrorDetailFromProperty(
	dst *extern.ErrorDetail,
	key string, value any,
) {
	code := key
	detail := fmt.Sprintf("%v", value)

	dst.Code = &code
	dst.Detail = detail
}
