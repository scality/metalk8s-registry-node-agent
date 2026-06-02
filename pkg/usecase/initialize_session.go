// nolint: dupl // normal to have the usecases very similar
package usecase

import (
	"log"
	"slices"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type InitializeSession struct {
	logger             *zerolog.Logger
	bucketManager      service.BucketManager
	archiveLister      service.ArchiveLister
	multipartStorer    service.MultipartStorer
	multipartInspector service.MultipartInspector
	bucketLocker       service.LockerUnlocker
}

func NewInitializeSession(
	logger *zerolog.Logger,
	bucketManager service.BucketManager,
	archiveLister service.ArchiveLister,
	multipartStorer service.MultipartStorer,
	multipartInspector service.MultipartInspector,
	bucketLocker service.LockerUnlocker,
) *InitializeSession {
	l := logger.With().Str("use_case", "initialize_session").Logger()

	return &InitializeSession{
		logger:             &l,
		bucketManager:      bucketManager,
		archiveLister:      archiveLister,
		multipartStorer:    multipartStorer,
		multipartInspector: multipartInspector,
		bucketLocker:       bucketLocker,
	}
}

func (uc *InitializeSession) Execute(solutionArchive *domain.SolutionArchive) (*domain.SessionStatus, error) {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Initializing session")

	// To avoid simultaneous uploads and deletions of the same solution archive
	uc.bucketLocker.Lock(solutionArchive)
	defer uc.bucketLocker.Unlock(solutionArchive)

	properties := solutionArchive.GetErrorProperties(
		"initialize_session",
		"",
	)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(217),
			errors.WithDetail("error on listing solution archives"),
			errors.WithProperties(properties),
		)
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
		return nil, errors.Wrap(err,
			errors.WithIdentifier(218),
			errors.WithDetail("error on checking if the session bucket already exists"),
			errors.WithProperties(properties),
			errors.WithProperty("bucket_name", bucketName),
		)
	}

	if exist {
		solutionArchiveStatus, err := uc.multipartInspector.GetMultipartFileStatus(bucketName, solutionArchive)
		if err != nil {
			return nil, errors.Wrap(err,
				errors.WithIdentifier(219),
				errors.WithDetail("error on getting multipart file status"),
				errors.WithProperties(properties),
				errors.WithProperty("bucket_name", bucketName),
			)
		}
		return &domain.SessionStatus{
			Version:                   solutionArchive.Version,
			IncompleteSolutionArchive: solutionArchiveStatus,
		}, nil
	}

	cleanup := func() {
		deleteBucket := func(bucketManager service.BucketManager, bucketName string) {
			if err := bucketManager.DeleteBucket(bucketName); err != nil {
				log.Println("Failed to delete the bucket", bucketName, err)
			}
		}
		deleteBucket(uc.bucketManager, bucketName)
	}

	// Create the session bucket with required working files (parts, metadata, recipient)
	err = uc.bucketManager.CreateBucket(bucketName)
	if err != nil {
		cleanup()
		return nil, errors.Wrap(err,
			errors.WithIdentifier(220),
			errors.WithDetail("error on creating session bucket"),
			errors.WithProperty("bucket_name", bucketName),
			errors.WithProperties(properties),
		)
	}

	solutionArchiveStatus, err := uc.multipartStorer.CreateMultipartFiles(bucketName, solutionArchive)
	if err != nil {
		cleanup()
		return nil, errors.Wrap(err,
			errors.WithIdentifier(221),
			errors.WithDetail("error on creating multipart files"),
			errors.WithProperties(properties),
			errors.WithProperty("bucket_name", bucketName),
		)
	}

	uc.logger.Debug().Msg("Session initialized")

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
	if err != nil && !errors.Is(err, domain.ErrBucketManagerNotFound) {
		return false, err
	}

	return slices.Contains(buckets, name), nil
}
