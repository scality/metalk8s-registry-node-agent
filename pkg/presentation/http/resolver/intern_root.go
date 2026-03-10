//nolint:ireturn // Generated code forces to return an interface.
package resolver

import (
	"context"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
)

type (
	GetSolutionArchive interface {
		DownloadSolutionArchive(
			ctx context.Context,
			request intern.DownloadSolutionArchiveRequestObject,
		) (intern.DownloadSolutionArchiveResponseObject, error)
	}

	HeadSolutionArchive interface {
		DescribeSolutionArchive(
			ctx context.Context,
			request intern.DescribeSolutionArchiveRequestObject,
		) (intern.DescribeSolutionArchiveResponseObject, error)
	}

	InternRoot struct {
		GetSolutionArchive
		HeadSolutionArchive
	}
)

var _ intern.StrictServerInterface = (*InternRoot)(nil)

func NewInternRoot(
	getSolutionArchive GetSolutionArchive,
	describeSolutionArchive HeadSolutionArchive,
) *InternRoot {
	return &InternRoot{
		GetSolutionArchive:  getSolutionArchive,
		HeadSolutionArchive: describeSolutionArchive,
	}
}

func (r *InternRoot) DownloadSolutionArchive(
	ctx context.Context,
	request intern.DownloadSolutionArchiveRequestObject,
) (intern.DownloadSolutionArchiveResponseObject, error) {
	response, err := r.GetSolutionArchive.DownloadSolutionArchive(ctx, request)
	if err != nil {
		return nil, errors.Intercept(err).
			WithDetail("failed to resolve download solution archive request").
			Throw()
	}

	return response, nil
}

func (r *InternRoot) DescribeSolutionArchive(
	ctx context.Context,
	request intern.DescribeSolutionArchiveRequestObject,
) (intern.DescribeSolutionArchiveResponseObject, error) {
	response, err := r.HeadSolutionArchive.DescribeSolutionArchive(ctx, request)
	if err != nil {
		return nil, errors.Intercept(err).
			WithDetail("failed to resolve describe solution archive request").
			Throw()
	}

	return response, nil
}
