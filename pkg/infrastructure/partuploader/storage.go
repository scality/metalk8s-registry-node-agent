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
	store service.StorageProvider,
	logger *zerolog.Logger,
	rootAPIPath string,
) *Storage {
	l := logger.With().Str("infrastructure", "partuploader").Logger()
	return &Storage{
		store:       store,
		logger:      &l,
		rootAPIPath: rootAPIPath,
	}
}

func (s *Storage) UploadPart(part *domain.Part) (*domain.SolutionArchiveStatus, error) {
	s.store.Lock()
	defer s.store.Unlock()

	// List all the buckets
	buckets, err := s.store.ListBuckets()
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
	solutionArchiveFromManifest, err := s.loadManifest(sessionBucket)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	// Test if a session related to the solution archive from the part exists
	if solutionArchiveFromManifest.Name != part.SolutionArchive.Name ||
		solutionArchiveFromManifest.Version != part.SolutionArchive.Version ||
		solutionArchiveFromManifest.Hash != part.SolutionArchive.Hash {
		return nil, errors.From(domain.ErrPartUploaderNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found in the current session manifest.").
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
			WithProperty("component", part.SolutionArchive.Name).
			WithProperty("version", part.SolutionArchive.Version).
			WithProperty("hash", part.SolutionArchive.Hash).
			Throw()
	}

	// When this is the first upload for a solution archive, the size is not set in the manifest
	// so we use the size from the part
	if solutionArchiveFromManifest.Size == 0 {
		solutionArchiveFromManifest.Size = part.SolutionArchive.Size

		err = s.store.DeleteMultipartFile(sessionBucket, solutionArchiveFromManifest)
		if err != nil {
			return nil, errors.Intercept(err).
				WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
				Throw()
		}

		_, err = s.store.CreateMultipartFiles(sessionBucket, solutionArchiveFromManifest)
		if err != nil {
			return nil, errors.Intercept(err).
				WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
				Throw()
		}
	}

	part.SolutionArchive = solutionArchiveFromManifest

	solutionArchiveStatus, err := s.store.WritePartToMultipartFile(sessionBucket, part)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	if !solutionArchiveStatus.IsComplete() {
		return solutionArchiveStatus, nil
	}

	// Solution archive upload is complete, so let's consolidate it,
	// move it to the storage root location and then
	// remove the bucket.
	cleanUpCorrupted := func() {
		if err := s.store.DeleteMultipartFile(sessionBucket, part.SolutionArchive); err != nil {
			s.logger.Error().Err(err).Any("solution archive", part.SolutionArchive).Msg("failed to delete multipart file")
		}

		if _, err := s.store.CreateMultipartFiles(sessionBucket, part.SolutionArchive); err != nil {
			s.logger.Error().Err(err).Any("solution archive", part.SolutionArchive).Msg("failed to create multipart file")
		}
	}

	err = s.store.ConsolidateMultipartFile(sessionBucket, part.SolutionArchive, library.FileSystemDefaultFileMode)
	if err != nil {
		cleanUpCorrupted()

		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(part.SolutionArchive)
	err = s.store.MoveFileToRoot(sessionBucket, part.SolutionArchive.Name, solutionArchiveFileName)
	if err != nil {
		cleanUpCorrupted()

		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}
	err = s.store.DeleteBucket(sessionBucket)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/uploads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	return solutionArchiveStatus, nil
}

// loadManifest loads the manifest from the session bucket.
func (s *Storage) loadManifest(sessionBucket string) (*domain.SolutionArchive, error) {
	solutionArchive, err := s.store.GetMultipartFile(sessionBucket)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return solutionArchive, nil
}
