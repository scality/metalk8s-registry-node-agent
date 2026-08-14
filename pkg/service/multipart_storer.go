package service

import (
	"context"
	"os"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

// MultipartStorer manages the multipart upload lifecycle: create, write
// parts, consolidate, and move the completed file to the root location.
type MultipartStorer interface {
	// CreateMultipartFiles creates into a bucket:
	// - a metadata file
	// - a multipart file recipient
	// - a parts synthesis file
	CreateMultipartFiles(
		ctx context.Context,
		bucketName string,
		solutionArchiveMeta *domain.SolutionArchive,
	) (*domain.SolutionArchiveStatus, error)

	// StorePart properly writes the content of the given part
	// into the recipient file on the bucket indicated by the given
	// bucketName.
	StorePart(
		ctx context.Context,
		bucketName string,
		solutionArchiveFromManifest *domain.SolutionArchive,
		part *domain.Part,
	) (*domain.SolutionArchiveStatus, error)

	// CommitPart properly updates the parts synthesis file
	// with the metadata of the given part.
	CommitPart(ctx context.Context, bucketName string, part *domain.Part) error

	// Consolidate consolidates all the parts of a multipart file
	// in a single flat file into the same bucket it is located and moves it to the root location.
	Consolidate(
		ctx context.Context,
		bucketName string,
		solutionArchiveMeta *domain.SolutionArchive,
		perm os.FileMode,
	) error
}
