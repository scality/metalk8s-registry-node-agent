package sessioninitializer

import (
	"errors"
	"log"
	"slices"

	"github.com/rs/zerolog"
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

func (s *Storage) InitializeSession(artifact *domain.Artifact) (*domain.SessionStatus, error) {
	s.store.Lock()
	defer s.store.Unlock()

	// List all artifacts in the storage
	// matching artifactStorageNamePattern
	fileNames, err := s.store.ListFiles()
	if err != nil {
		return nil, domain.Stamp(err)
	}

	// Check if the artifact already exists in the storage
	if library.ArtifactExists(artifact, fileNames) {
		return &domain.SessionStatus{
			Version:            artifact.Version,
			IncompleteArtifact: nil,
		}, nil
	}

	bucketName := library.GenBucketName(artifact)

	// Check if the session bucket already exists
	exist, err := s.sessionBucketAlreadyExists(bucketName)
	if err != nil {
		return nil, err
	}
	if exist {
		artifactStatus, err := s.store.GetMultipartFileStatus(bucketName, artifact)
		if err != nil {
			return nil, domain.Stamp(err)
		}
		return &domain.SessionStatus{
			Version:            artifact.Version,
			IncompleteArtifact: artifactStatus,
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
		return nil, domain.Stamp(err)
	}

	storageProvider := s.store
	artifactStatus, err := storageProvider.CreateMultipartFiles(bucketName, artifact)
	if err != nil {
		cleanup()
		return nil, domain.Stamp(err)
	}

	return &domain.SessionStatus{
		Version:            artifact.Version,
		IncompleteArtifact: artifactStatus,
	}, nil
}

// sessionBucketAlreadyExists returns if the session bucket already
// exists in the storage with the same prefixed name
// or an error when multiple session buckets are found.
func (s *Storage) sessionBucketAlreadyExists(name string) (bool, error) {
	buckets, err := s.store.ListBuckets()
	if err != nil && !errors.Is(err, domain.ErrSessionInitializerNotFoundError) {
		return false, domain.Stamp(err)
	}

	return slices.Contains(buckets, name), nil
}
