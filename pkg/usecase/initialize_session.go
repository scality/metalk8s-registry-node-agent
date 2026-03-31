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
	logger *zerolog.Logger
	store  service.StorageProvider
}

func NewInitializeSession(
	logger *zerolog.Logger,
	store service.StorageProvider,
) *InitializeSession {
	l := logger.With().Str("use_case", "initialize_session").Logger()

	return &InitializeSession{
		logger: &l,
		store:  store,
	}
}

func (uc *InitializeSession) Execute(solutionArchive *domain.SolutionArchive) (*domain.SessionStatus, error) {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Initializing session")
	uc.store.Lock()
	defer uc.store.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.store.ListFiles()
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
		solutionArchiveStatus, err := uc.store.GetMultipartFileStatus(bucketName, solutionArchive)
		if err != nil {
			return nil, errors.Stamp(err)
		}
		return &domain.SessionStatus{
			Version:                   solutionArchive.Version,
			IncompleteSolutionArchive: solutionArchiveStatus,
		}, nil
	}

	cleanup := func() {
		deleteBucket := func(storage service.StorageProvider, bucketName string) {
			if err := storage.DeleteBucket(bucketName); err != nil {
				log.Println("Failed to delete the bucket", bucketName, err)
			}
		}
		deleteBucket(uc.store, bucketName)
	}

	// Create the session bucket with required working files (parts, metadata, recipient)
	err = uc.store.CreateBucket(bucketName)
	if err != nil {
		cleanup()
		return nil, errors.Stamp(err)
	}

	storageProvider := uc.store
	solutionArchiveStatus, err := storageProvider.CreateMultipartFiles(bucketName, solutionArchive)
	if err != nil {
		cleanup()
		return nil, errors.Stamp(err)
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
	buckets, err := uc.store.ListBuckets()
	if err != nil && !errors.Is(err,
		errors.Intercept(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			Throw()) {
		return false, errors.Stamp(err)
	}

	return slices.Contains(buckets, name), nil
}
