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
	multipartRemover   service.MultipartRemover
	bucketLocker       service.LockerUnlocker
	archiveLocker      service.LockerUnlocker
	rootAPIPath        string
}

func NewUploadPart(
	logger *zerolog.Logger,
	bucketManager service.BucketManager,
	multipartUploader service.MultipartUploader,
	multipartInspector service.MultipartInspector,
	multipartRemover service.MultipartRemover,
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
		multipartRemover:   multipartRemover,
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
	solutionArchiveFromManifest, err := uc.multipartInspector.GetMultipartFile(sessionBucket)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	// Test if a session related to the solution archive from the part exists
	if solutionArchiveFromManifest.Name != part.SolutionArchive.Name ||
		solutionArchiveFromManifest.Version != part.SolutionArchive.Version {
		return nil, errors.From(domain.ErrPartUploaderNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found in the current session manifest").
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			WithProperty("component", part.SolutionArchive.Name).
			WithProperty("version", part.SolutionArchive.Version).
			Throw()
	}

	// When this is the first stored part for a solution archive, the size is not set in the manifest
	// so we use the size from the part
	if solutionArchiveFromManifest.Size == 0 {
		solutionArchiveFromManifest.Size = part.SolutionArchive.Size

		if err := uc.multipartRemover.DeleteMultipartFile(sessionBucket, solutionArchiveFromManifest); err != nil {
			return nil, errors.Intercept(err).
				WithProperty("instance", fmt.Sprintf("%s/uploads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
				Throw()
		}

		if _, err := uc.multipartUploader.CreateMultipartFiles(sessionBucket, solutionArchiveFromManifest); err != nil {
			return nil, errors.Intercept(err).
				WithProperty("instance", fmt.Sprintf("%s/uploads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
				Throw()
		}
	}

	part.SolutionArchive = solutionArchiveFromManifest

	solutionArchiveStatus, err := uc.multipartUploader.WritePartToRecipientFile(sessionBucket, part)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	err = uc.multipartUploader.CommitPart(sessionBucket, part)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	if !solutionArchiveStatus.IsComplete() {
		return solutionArchiveStatus, nil
	}

	// Solution archive is complete, so let's consolidate it,
	// move it to the storage root location and then
	// remove the bucket.
	cleanUpCorrupted := func() {
		if err := uc.multipartRemover.DeleteMultipartFile(sessionBucket, part.SolutionArchive); err != nil {
			uc.logger.Error().Err(err).Any("solution archive", part.SolutionArchive).Msg("failed to delete multipart file")
		}

		if _, err := uc.multipartUploader.CreateMultipartFiles(sessionBucket, part.SolutionArchive); err != nil {
			uc.logger.Error().Err(err).Any("solution archive", part.SolutionArchive).Msg("failed to create multipart file")
		}
	}

	err = uc.multipartUploader.ConsolidateMultipartFile(
		sessionBucket,
		part.SolutionArchive,
		library.FileSystemDefaultFileMode,
	)
	if err != nil {
		cleanUpCorrupted()
		// return nil, errors.Stamp(err)
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(part.SolutionArchive)
	err = uc.multipartUploader.MoveFileToRoot(sessionBucket, part.SolutionArchive.Name, solutionArchiveFileName)
	if err != nil {
		cleanUpCorrupted()
		// return nil, errors.Stamp(err)
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}
	err = uc.bucketManager.DeleteBucket(sessionBucket)
	if err != nil {
		// return nil, errors.Stamp(err)
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	uc.logger.Info().Msg("Part uploaded")

	return solutionArchiveStatus, nil
}
