package partuploader

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type Storage struct {
	store       service.StorageProvider
	logger      *zerolog.Logger
	rootAPIPath string
}

func NewStorage(
	artifact service.StorageProvider,
	logger *zerolog.Logger,
	rootAPIPath string,
) *Storage {
	l := logger.With().Str("infrastructure", "partuploader").Logger()
	return &Storage{
		store:       artifact,
		logger:      &l,
		rootAPIPath: rootAPIPath,
	}
}

func (s *Storage) UploadPart(part *domain.Part) (*domain.ArtifactStatus, error) {
	s.store.Lock()
	defer s.store.Unlock()

	// List all the buckets
	buckets, err := s.store.ListBuckets()
	if err != nil {
		return nil, errors.Stamp(err)
	}

	// Extract the session bucket
	sessionBucket, err := library.ExtractSessionBucket(buckets, part.Artifact)
	if err != nil {
		return nil, errors.Intercept(err).
			WithDetail("failed to extract the session bucket").
			Throw()
	}

	// Load the manifest from metadata file
	artifactFromManifest, err := s.loadManifest(sessionBucket)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
			Throw()
	}

	// Test if a session related to the artifact from the part exists
	if artifactFromManifest.Name != part.Artifact.Name ||
		artifactFromManifest.Version != part.Artifact.Version ||
		artifactFromManifest.Hash != part.Artifact.Hash {
		return nil, errors.From(domain.ErrPartUploaderNotFound).
			WithIdentifier(404000).
			WithDetail("Artifact not found in the current session manifest.").
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
			WithProperty("component", part.Artifact.Name).
			WithProperty("version", part.Artifact.Version).
			WithProperty("hash", part.Artifact.Hash).
			Throw()
	}

	// When this is the first upload for an artifact, the size is not set in the manifest
	// so we use the size from the part
	if artifactFromManifest.Size == 0 {
		artifactFromManifest.Size = part.Artifact.Size

		err = s.store.DeleteMultipartFile(sessionBucket, artifactFromManifest)
		if err != nil {
			return nil, errors.Intercept(err).
				WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
				Throw()
		}

		_, err = s.store.CreateMultipartFiles(sessionBucket, artifactFromManifest)
		if err != nil {
			return nil, errors.Intercept(err).
				WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
				Throw()
		}
	}

	part.Artifact = artifactFromManifest

	artifactStatus, err := s.store.WritePartToMultipartFile(sessionBucket, part)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
			Throw()
	}

	if !artifactStatus.IsComplete() {
		return artifactStatus, nil
	}

	// Artifact upload is complete, so let's consolidate it,
	// move it to the storage root location and then
	// remove the bucket.
	cleanUpCorrupted := func() {
		if err := s.store.DeleteMultipartFile(sessionBucket, part.Artifact); err != nil {
			s.logger.Error().Err(err).Any("artifact", part.Artifact).Msg("failed to delete multipart file")
		}

		if _, err := s.store.CreateMultipartFiles(sessionBucket, part.Artifact); err != nil {
			s.logger.Error().Err(err).Any("artifact", part.Artifact).Msg("failed to create multipart file")
		}
	}

	err = s.store.ConsolidateMultipartFile(sessionBucket, part.Artifact, library.FileSystemDefaultFileMode)
	if err != nil {
		cleanUpCorrupted()

		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
			Throw()
	}

	artifactFileName := library.GenArtifactFileName(part.Artifact)
	err = s.store.MoveFileToRoot(sessionBucket, part.Artifact.Name, artifactFileName)
	if err != nil {
		cleanUpCorrupted()

		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
			Throw()
	}
	err = s.store.DeleteBucket(sessionBucket)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
			Throw()
	}

	return artifactStatus, nil
}

// loadManifest loads the manifest from the session bucket.
func (s *Storage) loadManifest(sessionBucket string) (*domain.Artifact, error) {
	artifact, err := s.store.GetMultipartFile(sessionBucket)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return artifact, nil
}
