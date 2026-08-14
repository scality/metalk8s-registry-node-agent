//nolint:lll,dupl // normal to have handlers very similar
package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/usecase"
)

type DescribeSolutionArchive struct {
	logger *slog.Logger

	uc *usecase.DescribeSolutionArchive
}

func NewDescribeSolutionArchive(
	logger *slog.Logger,
	uc *usecase.DescribeSolutionArchive,
) *DescribeSolutionArchive {
	return &DescribeSolutionArchive{
		logger: logger.With(slog.String("http_handler", "describe_solution_archive")),
		uc:     uc,
	}
}

//nolint:ireturn // Generated code forces to return an interface.
func (h *DescribeSolutionArchive) DescribeSolutionArchive(
	ctx context.Context,
	request intern.DescribeSolutionArchiveRequestObject,
) (intern.DescribeSolutionArchiveResponseObject, error) {
	var solutionArchive domain.SolutionArchive

	if err := fillSolutionArchiveFromDescribeSolutionArchiveRequestObject(&solutionArchive, &request); err != nil {
		return h.genDescribeSolutionArchiveResponseObjectFromError(
			ctx,
			errors.Wrap(err,
				errors.WithIdentifier(http.StatusBadRequest),
			),
		)
	}

	size, err := h.uc.Execute(ctx, &solutionArchive)
	if err != nil {
		// Default value for the API error
		apiErr := errors.Wrap(err,
			errors.WithIdentifier(http.StatusInternalServerError),
		)
		if errors.IdentifierStartsWith(err, "197") {
			apiErr = errors.Wrap(err, errors.WithIdentifier(http.StatusNotFound))
		}

		return h.genDescribeSolutionArchiveResponseObjectFromError(ctx, apiErr)
	}

	return intern.DescribeSolutionArchive200Response{
		Headers: intern.DescribeSolutionArchive200ResponseHeaders{
			ContentLength: size,
		},
	}, nil
}

// genDescribeSolutionArchiveResponseObjectFromError generates a proper
// DescribeSolutionArchiveResponseObject from a given error.
//
//nolint:funlen,ireturn
func (h *DescribeSolutionArchive) genDescribeSolutionArchiveResponseObjectFromError(
	ctx context.Context,
	err error,
) (intern.DescribeSolutionArchiveResponseObject, error) {
	var apiErr *errors.Error

	var problemDetails intern.ProblemDetails

	// no need to test for bad conversion
	// the error is an underlying *errors.Error, wrapped in DescribeSolutionArchive
	errors.As(err, &apiErr) // nolint: errcheck

	h.fillProblemDetailsFromAPIErrorsError(ctx, &problemDetails, apiErr)

	switch int(*problemDetails.Status) {
	case http.StatusBadRequest:
		return intern.DescribeSolutionArchive400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: intern.BadRequestApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusUnauthorized:
		return intern.DescribeSolutionArchive401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: intern.UnauthorizedApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusForbidden:
		return intern.DescribeSolutionArchive403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: intern.ForbiddenApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusNotFound:
		return intern.DescribeSolutionArchive404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: intern.NotFoundApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	case http.StatusInternalServerError:
		return intern.DescribeSolutionArchive500ApplicationProblemPlusJSONResponse{
			ServerErrorApplicationProblemPlusJSONResponse: intern.ServerErrorApplicationProblemPlusJSONResponse(
				problemDetails,
			),
		}, nil

	default:
		return nil, errors.Wrap(err)
	}
}
