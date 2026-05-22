//nolint:lll,dupl // normal to have handlers very similar
package handler

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/usecase"
)

type DescribeSolutionArchive struct {
	logger *zerolog.Logger

	uc *usecase.DescribeSolutionArchive
}

func NewDescribeSolutionArchive(
	logger *zerolog.Logger,
	uc *usecase.DescribeSolutionArchive,
) *DescribeSolutionArchive {
	l := logger.With().Str("http_handler", "describe_solution_archive").Logger()

	return &DescribeSolutionArchive{
		logger: &l,
		uc:     uc,
	}
}

//nolint:ireturn // Generated code forces to return an interface.
func (h *DescribeSolutionArchive) DescribeSolutionArchive(
	_ context.Context,
	request intern.DescribeSolutionArchiveRequestObject,
) (intern.DescribeSolutionArchiveResponseObject, error) {
	var solutionArchive domain.SolutionArchive

	if err := fillSolutionArchiveFromDescribeSolutionArchiveRequestObject(&solutionArchive, &request); err != nil {
		return h.genDescribeSolutionArchiveResponseObjectFromError(
			errors.Wrap(err,
				errors.WithIdentifier(http.StatusBadRequest),
			),
		)
	}

	size, err := h.uc.Execute(&solutionArchive)
	if err != nil {
		// Default value for the API error
		apiErr := errors.Wrap(err,
			errors.WithIdentifier(http.StatusInternalServerError),
		)
		if errors.Is(err, errors.Wrap(domain.ErrNotFound, errors.WithIdentifier(404000))) {
			apiErr = errors.Wrap(err, errors.WithIdentifier(http.StatusNotFound))
		}

		return h.genDescribeSolutionArchiveResponseObjectFromError(apiErr)
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
	err error,
) (intern.DescribeSolutionArchiveResponseObject, error) {
	var apiErr *errors.Error

	var problemDetails intern.ProblemDetails

	errors.As(err, &apiErr)

	h.fillProblemDetailsFromAPIErrorsError(&problemDetails, apiErr)

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
