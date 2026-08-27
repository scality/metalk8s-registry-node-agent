package usecase

import (
	"context"
	"log/slog"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type ReceivePart struct {
	logger             *slog.Logger
	bucketManager      service.BucketManager
	multipartStorer    service.MultipartStorer
	multipartInspector service.MultipartInspector
	bucketLocker       service.LockerUnlocker
	archiveLocker      service.LockerUnlocker
	rootAPIPath        string
}

func NewReceivePart(
	logger *slog.Logger,
	bucketManager service.BucketManager,
	multipartStorer service.MultipartStorer,
	multipartInspector service.MultipartInspector,
	bucketLocker service.LockerUnlocker,
	archiveLocker service.LockerUnlocker,
	rootAPIPath string,
) *ReceivePart {
	return &ReceivePart{
		logger:             logger.With(slog.String("use_case", "receive_part")),
		bucketManager:      bucketManager,
		multipartStorer:    multipartStorer,
		multipartInspector: multipartInspector,
		bucketLocker:       bucketLocker,
		archiveLocker:      archiveLocker,
		rootAPIPath:        rootAPIPath,
	}
}

func (uc *ReceivePart) Execute(ctx context.Context, part *domain.Part) (*domain.SolutionArchiveStatus, error) {
	uc.logger.InfoContext(ctx, "Receiving part")

	uc.bucketLocker.Lock(part.SolutionArchive)
	defer uc.bucketLocker.Unlock(part.SolutionArchive)
	uc.archiveLocker.Lock(part.SolutionArchive)
	defer uc.archiveLocker.Unlock(part.SolutionArchive)

	properties := part.SolutionArchive.GetErrorProperties(
		"receive_part",
		part.SolutionArchive.GetUploadURL(uc.rootAPIPath),
	)

	// List all the buckets
	buckets, err := uc.bucketManager.ListBuckets()
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(232),
			errors.WithDetail("error on listing buckets"),
			errors.WithProperties(properties),
		)
	}

	// Extract the session bucket
	sessionBucket, err := library.ExtractSessionBucket(buckets, part.SolutionArchive)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(233),
			errors.WithDetail("failed to extract the session bucket"),
			errors.WithProperties(properties),
		)
	}

	// Load the manifest from metadata file
	solutionArchiveFromManifest, err := uc.multipartInspector.GetMultipartFile(ctx, sessionBucket)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(234),
			errors.WithDetail("error on retrieving the manifest file"),
			errors.WithProperties(properties),
		)
	}

	// Test if a session related to the solution archive from the part exists
	if solutionArchiveFromManifest.Name != part.SolutionArchive.Name ||
		solutionArchiveFromManifest.Version != part.SolutionArchive.Version {
		return nil, errors.Wrap(domain.ErrPartReceiverNotFound,
			errors.WithIdentifier(235),
			errors.WithDetail("solution archive not found in the current session manifest"),
			errors.WithProperties(properties),
		)
	}

	solutionArchiveStatus, err := uc.multipartStorer.StorePart(
		ctx,
		sessionBucket,
		solutionArchiveFromManifest,
		part,
	)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(236),
			errors.WithDetail("error on storing the part"),
			errors.WithProperties(properties),
		)
	}

	err = uc.multipartStorer.CommitPart(ctx, sessionBucket, part)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(237),
			errors.WithDetail("error on committing the part"),
			errors.WithProperties(properties),
		)
	}

	if !solutionArchiveStatus.IsComplete() {
		return solutionArchiveStatus, nil
	}

	// Solution archive is complete, so let's consolidate it,
	// move it to the storage root location and then
	// remove the bucket.
	err = uc.multipartStorer.Consolidate(
		ctx,
		sessionBucket,
		solutionArchiveFromManifest,
		library.FileSystemDefaultFileMode,
	)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(238),
			errors.WithDetail("error on consolidating the part"),
			errors.WithProperties(properties),
		)
	}

	uc.logger.InfoContext(ctx, "Part received")

	return solutionArchiveStatus, nil
}
