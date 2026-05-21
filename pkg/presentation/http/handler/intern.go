// nolint: dupl // normal to have the internal and external handlers very similar
package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
)

// fillSolutionArchiveFromDownloadSolutionArchiveRequestObject fills the Solution Archive object
// from the DownloadSolutionArchiveRequestObject object.
func fillSolutionArchiveFromDownloadSolutionArchiveRequestObject(
	dst *domain.Part,
	src *intern.DownloadSolutionArchiveRequestObject,
) error {

	if src.SolutionArchive == "" {
		return errors.From(domain.ErrHandlerMissingRequestParameter).
			WithDetail("parameter 'solution-archive' is missing").
			Throw()
	}

	if src.Version == "" {
		return errors.From(domain.ErrHandlerMissingRequestParameter).
			WithDetail("parameter 'version' is missing").
			Throw()
	}
	if src.Params.Range == "" {
		return errors.From(domain.ErrHandlerMissingRequestHeader).
			WithDetail("header 'Range' is missing").
			Throw()
	}
	start, end, _, err := parseRange(src.Params.Range)
	if err != nil {
		return errors.Stamp(err)
	}

	solutionArchive := &domain.SolutionArchive{
		Name:    src.SolutionArchive,
		Version: src.Version,
	}

	dst.SolutionArchive = solutionArchive
	dst.Meta = &domain.PartMeta{
		Start: start,
		End:   end,
	}

	return nil
}

// fillSolutionArchiveFromDescribeSolutionArchiveRequestObject fills the Solution Archive object
// from the DescribeSolutionArchiveRequestObject object.
func fillSolutionArchiveFromDescribeSolutionArchiveRequestObject(
	dst *domain.SolutionArchive,
	src *intern.DescribeSolutionArchiveRequestObject,
) error {
	if src.SolutionArchive == "" {
		return errors.From(domain.ErrHandlerMissingRequestParameter).
			WithDetail("parameter 'solution-archive' is missing").
			Throw()
	}

	if src.Version == "" {
		return errors.From(domain.ErrHandlerMissingRequestParameter).
			WithDetail("parameter 'version' is missing").
			Throw()
	}

	dst.Name = src.SolutionArchive
	dst.Version = src.Version

	return nil
}

// fillInternProblemDetailsFromAPIErrorsError fills the ProblemDetails object from the apierrors.Error object.
func fillInternProblemDetailsFromAPIErrorsError(
	ctx context.Context,
	logger *slog.Logger,
	dst *intern.ProblemDetails,
	src *errors.Error,
	logMsg string,
) {
	logger.ErrorContext(ctx, logMsg, slog.Any("error", src))

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

		errs := make(intern.Errors, 0, len(src.Properties))
		fillInternErrorDetailsFromPropertiesMap(&errs, src.Properties)

		dst.Errors = &errs
	}
}

func (h *DownloadSolutionArchive) fillProblemDetailsFromAPIErrorsError(
	ctx context.Context,
	dst *intern.ProblemDetails,
	src *errors.Error,
) {
	fillInternProblemDetailsFromAPIErrorsError(ctx, h.logger, dst, src, "Downloads API error")
}

func (h *DescribeSolutionArchive) fillProblemDetailsFromAPIErrorsError(
	ctx context.Context,
	dst *intern.ProblemDetails,
	src *errors.Error,
) {
	fillInternProblemDetailsFromAPIErrorsError(ctx, h.logger, dst, src, "Describe API error")
}

// fillInternErrorDetailsFromPropertiesMap fills the ErrorDetail slice from a map[string]any.
func fillInternErrorDetailsFromPropertiesMap(
	dst *intern.Errors,
	src map[string]any,
) {
	*dst = make([]intern.ErrorDetail, 0, len(src))

	for key, value := range src {
		if strings.HasPrefix(key, "problem_") {
			var errorDetail intern.ErrorDetail

			fillInternErrorDetailFromProperty(&errorDetail, key, value)

			*dst = append(*dst, errorDetail)
		}
	}
}

// fillInternErrorDetailFromProperty fills the ErrorDetail object from a property name and value.
func fillInternErrorDetailFromProperty(
	dst *intern.ErrorDetail,
	key string, value any,
) {
	code := key
	detail := fmt.Sprintf("%v", value)

	dst.Code = &code
	dst.Detail = detail
}
