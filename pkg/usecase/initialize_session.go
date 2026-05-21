// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"context"
	"log/slog"
	"slices"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type InitializeSession struct {
	logger             *slog.Logger
	bucketManager      service.BucketManager
	archiveLister      service.ArchiveLister
	multipartUploader  service.MultipartUploader
	multipartInspector service.MultipartInspector
	bucketLocker       service.LockerUnlocker
}

func NewInitializeSession(
	logger *slog.Logger,
	bucketManager service.BucketManager,
	archiveLister service.ArchiveLister,
	multipartUploader service.MultipartUploader,
	multipartInspector service.MultipartInspector,
	bucketLocker service.LockerUnlocker,
) *InitializeSession {
	return &InitializeSession{
		logger:             logger.With(slog.String("use_case", "initialize_session")),
		bucketManager:      bucketManager,
		archiveLister:      archiveLister,
		multipartUploader:  multipartUploader,
		multipartInspector: multipartInspector,
		bucketLocker:       bucketLocker,
	}
}

func (uc *InitializeSession) Execute(
	ctx context.Context, solutionArchive *domain.SolutionArchive,
) (*domain.SessionStatus, error) {
	uc.logger.DebugContext(ctx, "Initializing session",
		slog.Any("solution_archive", solutionArchive),
	)

	// To avoid simultaneous uploads and deletions of the same solution archive
	uc.bucketLocker.Lock(solutionArchive)
	defer uc.bucketLocker.Unlock(solutionArchive)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return nil, errors.Stamp(err)
	}

	// Check if the solution archive already exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		return &domain.SessionStatus{
			Version:                   solutionArchive.Version,
			IncompleteSolutionArchive: nil,
		}, nil
	}

	bucketName := library.GenBucketName(solutionArchive)

	// Check if the session bucket already exists
	exist, err := uc.sessionBucketAlreadyExists(bucketName)
	if err != nil {
		return nil, err
	}

	if exist {
		solutionArchiveStatus, err := uc.multipartInspector.GetMultipartFileStatus(ctx, bucketName, solutionArchive)
		if err != nil {
			return nil, errors.Stamp(err)
		}
		return &domain.SessionStatus{
			Version:                   solutionArchive.Version,
			IncompleteSolutionArchive: solutionArchiveStatus,
		}, nil
	}

	cleanup := func() {
		deleteBucket := func(bucketManager service.BucketManager, bucketName string) {
			if err := bucketManager.DeleteBucket(bucketName); err != nil {
				uc.logger.ErrorContext(ctx, "Failed to delete the bucket",
					slog.String("bucket_name", bucketName),
					slog.Any("error", err),
				)
			}
		}
		deleteBucket(uc.bucketManager, bucketName)
	}

	// Create the session bucket with required working files (parts, metadata, recipient)
	err = uc.bucketManager.CreateBucket(bucketName)
	if err != nil {
		cleanup()
		return nil, errors.Stamp(err)
	}

	solutionArchiveStatus, err := uc.multipartUploader.CreateMultipartFiles(ctx, bucketName, solutionArchive)
	if err != nil {
		cleanup()
		return nil, errors.Stamp(err)
	}

	uc.logger.DebugContext(ctx, "Session initialized")

	return &domain.SessionStatus{
		Version:                   solutionArchive.Version,
		IncompleteSolutionArchive: solutionArchiveStatus,
	}, nil
}

// sessionBucketAlreadyExists returns if the session bucket already
// exists in the storage with the same prefixed name
// or an error when multiple session buckets are found.
func (uc *InitializeSession) sessionBucketAlreadyExists(name string) (bool, error) {
	buckets, err := uc.bucketManager.ListBuckets()
	if err != nil && !errors.Is(err,
		errors.Intercept(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			Throw()) {
		return false, errors.Stamp(err)
	}

	return slices.Contains(buckets, name), nil
}
