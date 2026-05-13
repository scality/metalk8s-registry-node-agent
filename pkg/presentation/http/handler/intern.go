// nolint: dupl // normal to have the internal and external handlers very similar
package handler

import (
	"strings"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
	"k8s.io/utils/ptr"
)

// fillSolutionArchiveFromDownloadSolutionArchiveRequestObject fills the Solution Archive object
// from the DownloadSolutionArchiveRequestObject object.
func fillSolutionArchiveFromDownloadSolutionArchiveRequestObject(
	dst *domain.Part,
	src *intern.DownloadSolutionArchiveRequestObject,
) error {

	if src.SolutionArchive == "" {
		return errors.Wrap(domain.ErrHandlerMissingRequestParameter,
			errors.WithIdentifier(400003),
			errors.WithDetail("parameter 'solution-archive' is missing"),
		)
	}

	if src.Version == "" {
		return errors.Wrap(domain.ErrHandlerMissingRequestParameter,
			errors.WithIdentifier(400003),
			errors.WithDetail("parameter 'version' is missing"),
		)
	}
	if src.Params.Range == "" {
		return errors.Wrap(domain.ErrHandlerMissingRequestHeader,
			errors.WithIdentifier(400002),
			errors.WithDetail("header 'Range' is missing"),
		)
	}
	start, end, _, err := library.ParseRange(src.Params.Range)
	if err != nil {
		return errors.Wrap(err)
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
		return errors.Wrap(domain.ErrHandlerMissingRequestParameter,
			errors.WithIdentifier(400003),
			errors.WithDetail("parameter 'solution-archive' is missing"),
		)
	}

	if src.Version == "" {
		return errors.Wrap(domain.ErrHandlerMissingRequestParameter,
			errors.WithIdentifier(400003),
			errors.WithDetail("parameter 'version' is missing"),
		)
	}

	dst.Name = src.SolutionArchive
	dst.Version = src.Version

	return nil
}

// fillInternProblemDetailsFromAPIErrorsError fills the ProblemDetails object from the apierrors.Error object.
func fillInternProblemDetailsFromAPIErrorsError(
	logger *zerolog.Logger,
	dst *intern.ProblemDetails,
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

func (h *DownloadSolutionArchive) fillProblemDetailsFromAPIErrorsError(
	dst *intern.ProblemDetails,
	src *errors.Error,
) {
	fillInternProblemDetailsFromAPIErrorsError(h.logger, dst, src, "Downloads API error")
}

func (h *DescribeSolutionArchive) fillProblemDetailsFromAPIErrorsError(
	dst *intern.ProblemDetails,
	src *errors.Error,
) {
	fillInternProblemDetailsFromAPIErrorsError(h.logger, dst, src, "Describe API error")
}
