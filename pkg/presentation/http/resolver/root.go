//nolint:ireturn // Generated code forces to return an interface.
package resolver

import (
	"context"

	"github.com/pkg/errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/generated"
)

type (
	UploadPart interface {
		UploadChunk(
			ctx context.Context,
			request generated.UploadChunkRequestObject,
		) (generated.UploadChunkResponseObject, error)
	}

	Root struct {
		UploadPart
	}
)

var _ generated.StrictServerInterface = (*Root)(nil)

func NewRoot(
	uploadPart UploadPart,
) *Root {
	return &Root{
		UploadPart: uploadPart,
	}
}

func (r *Root) UploadChunk(
	ctx context.Context,
	request generated.UploadChunkRequestObject,
) (generated.UploadChunkResponseObject, error) {
	response, err := r.UploadPart.UploadChunk(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve upload chunk request")
	}

	return response, nil
}
