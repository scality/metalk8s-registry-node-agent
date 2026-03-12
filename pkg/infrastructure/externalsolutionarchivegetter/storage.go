package externalsolutionarchivegetter

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type Storage struct {
	store              service.StorageProvider
	logger             *zerolog.Logger
	externalDownloader service.ExternalDownloader
	rootAPIPath        string
	chunkSize          int64
}

func NewStorage(
	store service.StorageProvider,
	logger *zerolog.Logger,
	externalDownloader service.ExternalDownloader,
	rootAPIPath string,
	chunkSize int64,
) *Storage {
	l := logger.With().Str("infrastructure", "external_solution_archive_getter").Logger()
	return &Storage{
		store:              store,
		logger:             &l,
		externalDownloader: externalDownloader,
		rootAPIPath:        rootAPIPath,
		chunkSize:          chunkSize,
	}
}

var _ service.ExternalSolutionArchiveGetter = &Storage{}

func (s *Storage) GetExternalSolutionArchive(solutionArchive *domain.SolutionArchive, downloadURL string) error {
	s.store.Lock()
	defer s.store.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := s.store.ListFiles()
	if err != nil {
		return errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		return nil
	}

	// Get description of the solution archive
	solutionArchiveSize, err := s.externalDownloader.GetDescription(downloadURL)
	if err != nil {
		return errors.Stamp(err)
	}

	// Iterate over the chunks
	numChunks := (solutionArchiveSize + s.chunkSize - 1) / s.chunkSize
	partSolutionArchive := &domain.SolutionArchive{
		Name:    solutionArchive.Name,
		Version: solutionArchive.Version,
		Hash:    solutionArchive.Hash,
		Size:    solutionArchiveSize,
	}
	for chunkIndex := range numChunks {
		start := chunkIndex * s.chunkSize
		end := min(start+s.chunkSize, solutionArchiveSize) - 1
		part := &domain.Part{
			SolutionArchive: partSolutionArchive,
			Meta: &domain.PartMeta{
				Start: start,
				End:   end,
			},
		}

		body, err := s.externalDownloader.Download(downloadURL, solutionArchive.Hash, start, end, solutionArchiveSize)
		if err != nil {
			return errors.Stamp(err)
		}

		// Store the chunk in the file (streams via io.Copy); then close the HTTP body this iteration.
		part.Content = body

		_, err = s.storePart(part)
		_ = body.Close() // nolint: errcheck // Return path uses storePart err; Close releases the connection.
		if err != nil {
			return errors.Stamp(err)
		}
	}

	return nil
}

func (s *Storage) storePart(part *domain.Part) (*domain.SolutionArchiveStatus, error) {
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
	solutionArchiveFromManifest, err := s.store.GetMultipartFile(sessionBucket)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	// Test if a session related to the solution archive from the part exists
	if solutionArchiveFromManifest.Name != part.SolutionArchive.Name ||
		solutionArchiveFromManifest.Version != part.SolutionArchive.Version ||
		solutionArchiveFromManifest.Hash != part.SolutionArchive.Hash {
		return nil, errors.From(domain.ErrPartUploaderNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found in the current session manifest").
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
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
				WithProperty("instance", fmt.Sprintf("%s/downloads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
				Throw()
		}

		_, err = s.store.CreateMultipartFiles(sessionBucket, solutionArchiveFromManifest)
		if err != nil {
			return nil, errors.Intercept(err).
				WithProperty("instance", fmt.Sprintf("%s/downloads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
				Throw()
		}
	}

	part.SolutionArchive = solutionArchiveFromManifest

	solutionArchiveStatus, err := s.store.WritePartToMultipartFile(sessionBucket, part)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
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
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(part.SolutionArchive)
	err = s.store.MoveFileToRoot(sessionBucket, part.SolutionArchive.Name, solutionArchiveFileName)
	if err != nil {
		cleanUpCorrupted()

		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}
	err = s.store.DeleteBucket(sessionBucket)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", s.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	return solutionArchiveStatus, nil
}
