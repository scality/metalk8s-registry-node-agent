package sessioninitializer

import (
	"log"
	"slices"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type Storage struct {
	store  service.StorageProvider
	logger *zerolog.Logger
}

func NewStorage(
	store service.StorageProvider,
	logger *zerolog.Logger,
) *Storage {
	l := logger.With().Str("infrastructure", "sessioninitializer").Logger()
	return &Storage{
		store:  store,
		logger: &l,
	}
}

func (s *Storage) InitializeSession(solutionArchive *domain.SolutionArchive) (*domain.SessionStatus, error) {
	s.store.Lock()
	defer s.store.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := s.store.ListFiles()
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
	exist, err := s.sessionBucketAlreadyExists(bucketName)
	if err != nil {
		return nil, err
	}
	if exist {
		solutionArchiveStatus, err := s.store.GetMultipartFileStatus(bucketName, solutionArchive)
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
		deleteBucket(s.store, bucketName)
	}

	// Create the session bucket with required working files (parts, metadata, recipient)
	err = s.store.CreateBucket(bucketName)
	if err != nil {
		cleanup()
		return nil, errors.Stamp(err)
	}

	storageProvider := s.store
	solutionArchiveStatus, err := storageProvider.CreateMultipartFiles(bucketName, solutionArchive)
	if err != nil {
		cleanup()
		return nil, errors.Stamp(err)
	}

	return &domain.SessionStatus{
		Version:                   solutionArchive.Version,
		IncompleteSolutionArchive: solutionArchiveStatus,
	}, nil
}

// sessionBucketAlreadyExists returns if the session bucket already
// exists in the storage with the same prefixed name
// or an error when multiple session buckets are found.
func (s *Storage) sessionBucketAlreadyExists(name string) (bool, error) {
	buckets, err := s.store.ListBuckets()
	if err != nil && !errors.Is(err,
		errors.Intercept(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			Throw()) {
		return false, errors.Stamp(err)
	}

	return slices.Contains(buckets, name), nil
}
