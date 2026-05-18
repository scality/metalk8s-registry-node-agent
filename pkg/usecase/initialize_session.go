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
	multipartUploader  service.MultipartUploader
	multipartInspector service.MultipartInspector
	bucketLocker       service.LockerUnlocker
}

func NewInitializeSession(
	logger *zerolog.Logger,
	bucketManager service.BucketManager,
	archiveLister service.ArchiveLister,
	multipartUploader service.MultipartUploader,
	multipartInspector service.MultipartInspector,
	bucketLocker service.LockerUnlocker,
) *InitializeSession {
	l := logger.With().Str("use_case", "initialize_session").Logger()

	return &InitializeSession{
		logger:             &l,
		bucketManager:      bucketManager,
		archiveLister:      archiveLister,
		multipartUploader:  multipartUploader,
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

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return nil, errors.Wrap(err)
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
		solutionArchiveStatus, err := uc.multipartInspector.GetMultipartFileStatus(bucketName, solutionArchive)
		if err != nil {
			return nil, errors.Wrap(err)
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
		return nil, errors.Wrap(err)
	}

	solutionArchiveStatus, err := uc.multipartUploader.CreateMultipartFiles(bucketName, solutionArchive)
	if err != nil {
		cleanup()
		return nil, errors.Wrap(err)
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
		return false, errors.Wrap(err)
	}

	return slices.Contains(buckets, name), nil
}
