package partuploader

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library/apierrors"
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
		return nil, apierrors.Stamp(err)
	}

	// Extract the session bucket
	sessionBucket, err := library.ExtractSessionBucket(buckets, part.Artifact)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract the session bucket")
	}

	// Load the manifest from metadata file
	artifactFromManifest, err := s.loadManifest(sessionBucket)
	if err != nil {
		return nil, apierrors.Intercept(err).
			AtInstance(fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
			Throw()
	}

	// Test if a session related to the artifact from the part exists
	if artifactFromManifest.Name != part.Artifact.Name ||
		artifactFromManifest.Version != part.Artifact.Version ||
		artifactFromManifest.Hash != part.Artifact.Hash {
		// FIXME This implementation should not know about HTTP errors
		return nil, apierrors.FromTemplate(apierrors.Err404000NotFound).
			WithDetail("Artifact not found in the current session manifest.").
			AtInstance(fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
			AddProperty("component", part.Artifact.Name).
			AddProperty("version", part.Artifact.Version).
			AddProperty("hash", part.Artifact.Hash).
			Throw()
	}

	// When this is the first upload for an artifact, the size is not set in the manifest
	// so we use the size from the part
	if artifactFromManifest.Size == 0 {
		artifactFromManifest.Size = part.Artifact.Size

		err = s.store.DeleteMultipartFile(sessionBucket, artifactFromManifest)
		if err != nil {
			return nil, apierrors.Intercept(err).
				AtInstance(fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
				Throw()
		}

		_, err = s.store.CreateMultipartFiles(sessionBucket, artifactFromManifest)
		if err != nil {
			return nil, apierrors.Intercept(err).
				AtInstance(fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
				Throw()
		}
	}

	part.Artifact = artifactFromManifest

	artifactStatus, err := s.store.WritePartToMultipartFile(sessionBucket, part)
	if err != nil {
		return nil, apierrors.Intercept(err).
			AtInstance(fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
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

		return nil, apierrors.Intercept(err).
			AtInstance(fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
			Throw()
	}

	artifactFileName := library.GenArtifactFileName(part.Artifact)
	err = s.store.MoveFileToRoot(sessionBucket, part.Artifact.Name, artifactFileName)
	if err != nil {
		cleanUpCorrupted()

		return nil, apierrors.Intercept(err).
			AtInstance(fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
			Throw()
	}
	err = s.store.DeleteBucket(sessionBucket)
	if err != nil {
		return nil, apierrors.Intercept(err).
			AtInstance(fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.Artifact.Name)).
			Throw()
	}

	return artifactStatus, nil
}

// loadManifest loads the manifest from the session bucket.
func (s *Storage) loadManifest(sessionBucket string) (*domain.Artifact, error) {
	artifact, err := s.store.GetMultipartFile(sessionBucket)
	if err != nil {
		return nil, apierrors.Stamp(err)
	}

	return artifact, nil
}
