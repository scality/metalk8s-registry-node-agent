package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type ReceivePart struct {
	logger             *zerolog.Logger
	bucketManager      service.BucketManager
	multipartStorer    service.MultipartStorer
	multipartInspector service.MultipartInspector
	bucketLocker       service.LockerUnlocker
	archiveLocker      service.LockerUnlocker
	rootAPIPath        string
}

func NewReceivePart(
	logger *zerolog.Logger,
	bucketManager service.BucketManager,
	multipartStorer service.MultipartStorer,
	multipartInspector service.MultipartInspector,
	bucketLocker service.LockerUnlocker,
	archiveLocker service.LockerUnlocker,
	rootAPIPath string,
) *ReceivePart {
	l := logger.With().Str("use_case", "receive_part").Logger()

	return &ReceivePart{
		logger:             &l,
		bucketManager:      bucketManager,
		multipartStorer:    multipartStorer,
		multipartInspector: multipartInspector,
		bucketLocker:       bucketLocker,
		archiveLocker:      archiveLocker,
		rootAPIPath:        rootAPIPath,
	}
}

func (uc *ReceivePart) Execute(part *domain.Part) (*domain.SolutionArchiveStatus, error) {
	uc.logger.Info().Msg("Receiving part")

	uc.bucketLocker.Lock(part.SolutionArchive)
	defer uc.bucketLocker.Unlock(part.SolutionArchive)
	uc.archiveLocker.Lock(part.SolutionArchive)
	defer uc.archiveLocker.Unlock(part.SolutionArchive)

	// List all the buckets
	buckets, err := uc.bucketManager.ListBuckets()
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(232),
			errors.WithDetail("error on listing buckets"),
			errors.WithProperty("usecase", "receive_part"),
			errors.WithProperty("solution_archive_name", part.SolutionArchive.Name),
			errors.WithProperty("solution_archive_version", part.SolutionArchive.Version),
			errors.WithProperty("instance", part.SolutionArchive.GetUploadURL(uc.rootAPIPath)),
		)
	}

	// Extract the session bucket
	sessionBucket, err := library.ExtractSessionBucket(buckets, part.SolutionArchive)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(233),
			errors.WithDetail("failed to extract the session bucket"),
			errors.WithProperty("usecase", "receive_part"),
			errors.WithProperty("solution_archive_name", part.SolutionArchive.Name),
			errors.WithProperty("solution_archive_version", part.SolutionArchive.Version),
			errors.WithProperty("instance", part.SolutionArchive.GetUploadURL(uc.rootAPIPath)),
		)
	}

	// Load the manifest from metadata file
	solutionArchiveFromManifest, err := uc.multipartInspector.GetMultipartFile(sessionBucket)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(234),
			errors.WithDetail("error on retrieving the manifest file"),
			errors.WithProperty("usecase", "receive_part"),
			errors.WithProperty("solution_archive_name", part.SolutionArchive.Name),
			errors.WithProperty("solution_archive_version", part.SolutionArchive.Version),
			errors.WithProperty("instance", part.SolutionArchive.GetUploadURL(uc.rootAPIPath)),
		)
	}

	// Test if a session related to the solution archive from the part exists
	if solutionArchiveFromManifest.Name != part.SolutionArchive.Name ||
		solutionArchiveFromManifest.Version != part.SolutionArchive.Version {
		return nil, errors.Wrap(domain.ErrPartReceiverNotFound,
			errors.WithIdentifier(235),
			errors.WithDetail("solution archive not found in the current session manifest"),
			errors.WithProperty("usecase", "receive_part"),
			errors.WithProperty("solution_archive_name", part.SolutionArchive.Name),
			errors.WithProperty("solution_archive_version", part.SolutionArchive.Version),
			errors.WithProperty("instance", part.SolutionArchive.GetUploadURL(uc.rootAPIPath)),
			errors.WithProperty("component", part.SolutionArchive.Name),
			errors.WithProperty("version", part.SolutionArchive.Version),
		)
	}

	solutionArchiveStatus, err := uc.multipartStorer.StorePart(
		sessionBucket,
		solutionArchiveFromManifest,
		part,
	)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(236),
			errors.WithDetail("error on storing the part"),
			errors.WithProperty("usecase", "receive_part"),
			errors.WithProperty("solution_archive_name", part.SolutionArchive.Name),
			errors.WithProperty("solution_archive_version", part.SolutionArchive.Version),
			errors.WithProperty("instance", part.SolutionArchive.GetUploadURL(uc.rootAPIPath)),
		)
	}

	err = uc.multipartStorer.CommitPart(sessionBucket, part)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(237),
			errors.WithDetail("error on committing the part"),
			errors.WithProperty("usecase", "receive_part"),
			errors.WithProperty("solution_archive_name", part.SolutionArchive.Name),
			errors.WithProperty("solution_archive_version", part.SolutionArchive.Version),
			errors.WithProperty("instance", part.SolutionArchive.GetUploadURL(uc.rootAPIPath)),
		)
	}

	if !solutionArchiveStatus.IsComplete() {
		return solutionArchiveStatus, nil
	}

	// Solution archive is complete, so let's consolidate it,
	// move it to the storage root location and then
	// remove the bucket.
	err = uc.multipartStorer.Consolidate(
		sessionBucket,
		solutionArchiveFromManifest,
		library.FileSystemDefaultFileMode,
	)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(238),
			errors.WithDetail("error on consolidating the part"),
			errors.WithProperty("usecase", "receive_part"),
			errors.WithProperty("solution_archive_name", part.SolutionArchive.Name),
			errors.WithProperty("solution_archive_version", part.SolutionArchive.Version),
			errors.WithProperty("instance", part.SolutionArchive.GetUploadURL(uc.rootAPIPath)),
		)
	}

	uc.logger.Info().Msg("Part received")

	return solutionArchiveStatus, nil
}
