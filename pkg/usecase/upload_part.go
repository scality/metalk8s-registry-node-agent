package usecase

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type UploadPart struct {
	logger             *zerolog.Logger
	bucketManager      service.BucketManager
	multipartUploader  service.MultipartUploader
	multipartInspector service.MultipartInspector
	bucketLocker       service.LockerUnlocker
	archiveLocker      service.LockerUnlocker
	rootAPIPath        string
}

func NewUploadPart(
	logger *zerolog.Logger,
	bucketManager service.BucketManager,
	multipartUploader service.MultipartUploader,
	multipartInspector service.MultipartInspector,
	bucketLocker service.LockerUnlocker,
	archiveLocker service.LockerUnlocker,
	rootAPIPath string,
) *UploadPart {
	l := logger.With().Str("use_case", "upload_part").Logger()

	return &UploadPart{
		logger:             &l,
		bucketManager:      bucketManager,
		multipartUploader:  multipartUploader,
		multipartInspector: multipartInspector,
		bucketLocker:       bucketLocker,
		archiveLocker:      archiveLocker,
		rootAPIPath:        rootAPIPath,
	}
}

func (uc *UploadPart) Execute(part *domain.Part) (*domain.SolutionArchiveStatus, error) {
	uc.logger.Info().Msg("Uploading part")

	uc.bucketLocker.Lock(part.SolutionArchive)
	defer uc.bucketLocker.Unlock(part.SolutionArchive)
	uc.archiveLocker.Lock(part.SolutionArchive)
	defer uc.archiveLocker.Unlock(part.SolutionArchive)

	// List all the buckets
	buckets, err := uc.bucketManager.ListBuckets()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	// Extract the session bucket
	sessionBucket, err := library.ExtractSessionBucket(buckets, part.SolutionArchive)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithDetail("failed to extract the session bucket"),
		)
	}

	// Load the manifest from metadata file
	solutionArchiveFromManifest, err := uc.multipartInspector.GetMultipartFile(sessionBucket)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/uploads/%s/%s",
				uc.rootAPIPath,
				part.SolutionArchive.Name,
				part.SolutionArchive.Version,
			)),
		)
	}

	// Test if a session related to the solution archive from the part exists
	if solutionArchiveFromManifest.Name != part.SolutionArchive.Name ||
		solutionArchiveFromManifest.Version != part.SolutionArchive.Version {
		return nil, errors.Wrap(domain.ErrPartUploaderNotFound,
			errors.WithIdentifier(404000),
			errors.WithDetail("solution archive not found in the current session manifest"),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/uploads/%s/%s",
				uc.rootAPIPath,
				part.SolutionArchive.Name,
				part.SolutionArchive.Version,
			)),
			errors.WithProperty("component", part.SolutionArchive.Name),
			errors.WithProperty("version", part.SolutionArchive.Version),
		)
	}

	solutionArchiveStatus, err := uc.multipartUploader.StorePart(
		sessionBucket,
		solutionArchiveFromManifest,
		part,
	)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/uploads/%s/%s",
				uc.rootAPIPath,
				part.SolutionArchive.Name,
				part.SolutionArchive.Version,
			)),
		)
	}

	err = uc.multipartUploader.CommitPart(sessionBucket, part)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/uploads/%s/%s",
				uc.rootAPIPath,
				part.SolutionArchive.Name,
				part.SolutionArchive.Version,
			)),
		)
	}

	if !solutionArchiveStatus.IsComplete() {
		return solutionArchiveStatus, nil
	}

	// Solution archive is complete, so let's consolidate it,
	// move it to the storage root location and then
	// remove the bucket.
	err = uc.multipartUploader.Consolidate(
		sessionBucket,
		solutionArchiveFromManifest,
		library.FileSystemDefaultFileMode,
	)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/uploads/%s/%s",
				uc.rootAPIPath,
				part.SolutionArchive.Name,
				part.SolutionArchive.Version,
			)),
		)
	}

	uc.logger.Info().Msg("Part uploaded")

	return solutionArchiveStatus, nil
}
