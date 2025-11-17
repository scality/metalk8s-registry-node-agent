//nolint:ireturn // Generated code forces to return an interface.
package resolver

import (
	"context"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
)

type (
	UploadPart interface {
		UploadChunk(
			ctx context.Context,
			request extern.UploadChunkRequestObject,
		) (extern.UploadChunkResponseObject, error)
	}

	ExternRoot struct {
		UploadPart
	}
)

var _ extern.StrictServerInterface = (*ExternRoot)(nil)

func NewExternRoot(
	uploadPart UploadPart,
) *ExternRoot {
	return &ExternRoot{
		UploadPart: uploadPart,
	}
}

func (r *ExternRoot) UploadChunk(
	ctx context.Context,
	request extern.UploadChunkRequestObject,
) (extern.UploadChunkResponseObject, error) {
	response, err := r.UploadPart.UploadChunk(ctx, request)
	if err != nil {
		return nil, errors.Intercept(err).
			WithDetail("failed to resolve upload chunk request").
			Throw()
	}

	return response, nil
}
