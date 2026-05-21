package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type UploadPart struct {
	logger             *slog.Logger
	bucketManager      service.BucketManager
	multipartUploader  service.MultipartUploader
	multipartInspector service.MultipartInspector
	bucketLocker       service.LockerUnlocker
	archiveLocker      service.LockerUnlocker
	rootAPIPath        string
}

func NewUploadPart(
	logger *slog.Logger,
	bucketManager service.BucketManager,
	multipartUploader service.MultipartUploader,
	multipartInspector service.MultipartInspector,
	bucketLocker service.LockerUnlocker,
	archiveLocker service.LockerUnlocker,
	rootAPIPath string,
) *UploadPart {
	return &UploadPart{
		logger:             logger.With(slog.String("use_case", "upload_part")),
		bucketManager:      bucketManager,
		multipartUploader:  multipartUploader,
		multipartInspector: multipartInspector,
		bucketLocker:       bucketLocker,
		archiveLocker:      archiveLocker,
		rootAPIPath:        rootAPIPath,
	}
}

func (uc *UploadPart) Execute(ctx context.Context, part *domain.Part) (*domain.SolutionArchiveStatus, error) {
	uc.logger.InfoContext(ctx, "Uploading part")

	uc.bucketLocker.Lock(part.SolutionArchive)
	defer uc.bucketLocker.Unlock(part.SolutionArchive)
	uc.archiveLocker.Lock(part.SolutionArchive)
	defer uc.archiveLocker.Unlock(part.SolutionArchive)

	// List all the buckets
	buckets, err := uc.bucketManager.ListBuckets()
	if err != nil {
		return nil, errors.Stamp(err)
	}

	// Extract the session bucket
	sessionBucket, err := library.ExtractSessionBucket(buckets, part.SolutionArchive)
	if err != nil {
		return nil, errors.Intercept(err).
			WithDetail("failed to extract the session bucket").
			Throw()
	}

	// Load the manifest from metadata file
	solutionArchiveFromManifest, err := uc.multipartInspector.GetMultipartFile(ctx, sessionBucket)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf(
				"%s/uploads/%s/%s",
				uc.rootAPIPath,
				part.SolutionArchive.Name,
				part.SolutionArchive.Version,
			)).
			Throw()
	}

	// Test if a session related to the solution archive from the part exists
	if solutionArchiveFromManifest.Name != part.SolutionArchive.Name ||
		solutionArchiveFromManifest.Version != part.SolutionArchive.Version {
		return nil, errors.From(domain.ErrPartUploaderNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found in the current session manifest").
			WithProperty("instance", fmt.Sprintf(
				"%s/uploads/%s/%s",
				uc.rootAPIPath,
				part.SolutionArchive.Name,
				part.SolutionArchive.Version,
			)).
			WithProperty("component", part.SolutionArchive.Name).
			WithProperty("version", part.SolutionArchive.Version).
			Throw()
	}

	solutionArchiveStatus, err := uc.multipartUploader.StorePart(
		ctx,
		sessionBucket,
		solutionArchiveFromManifest,
		part,
	)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf(
				"%s/uploads/%s/%s",
				uc.rootAPIPath,
				part.SolutionArchive.Name,
				part.SolutionArchive.Version,
			)).
			Throw()
	}

	err = uc.multipartUploader.CommitPart(ctx, sessionBucket, part)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf(
				"%s/uploads/%s/%s",
				uc.rootAPIPath,
				part.SolutionArchive.Name,
				part.SolutionArchive.Version,
			)).
			Throw()
	}

	if !solutionArchiveStatus.IsComplete() {
		return solutionArchiveStatus, nil
	}

	// Solution archive is complete, so let's consolidate it,
	// move it to the storage root location and then
	// remove the bucket.
	err = uc.multipartUploader.Consolidate(
		ctx,
		sessionBucket,
		solutionArchiveFromManifest,
		library.FileSystemDefaultFileMode,
	)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf(
				"%s/uploads/%s/%s",
				uc.rootAPIPath,
				part.SolutionArchive.Name,
				part.SolutionArchive.Version,
			)).
			Throw()
	}

	uc.logger.InfoContext(ctx, "Part uploaded")

	return solutionArchiveStatus, nil
}
