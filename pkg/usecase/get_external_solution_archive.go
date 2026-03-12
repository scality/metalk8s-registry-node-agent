package usecase

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type GetExternalSolutionArchive struct {
	logger             *zerolog.Logger
	externalDownloader service.ExternalDownloader
	bucketManager      service.BucketManager
	archiveLister      service.ArchiveLister
	archiveLocker      service.LockerUnlocker
	multipartUploader  service.MultipartUploader
	multipartInspector service.MultipartInspector
	multipartRemover   service.MultipartRemover
	rootAPIPath        string
	chunkSize          int64
}

func NewGetExternalSolutionArchive(
	logger *zerolog.Logger,
	externalDownloader service.ExternalDownloader,
	bucketManager service.BucketManager,
	archiveLister service.ArchiveLister,
	archiveLocker service.LockerUnlocker,
	multipartUploader service.MultipartUploader,
	multipartInspector service.MultipartInspector,
	multipartRemover service.MultipartRemover,
	rootAPIPath string,
	chunkSize int64,
) *GetExternalSolutionArchive {
	l := logger.With().Str("use_case", "get_external_solution_archive").Logger()

	return &GetExternalSolutionArchive{
		logger:             &l,
		externalDownloader: externalDownloader,
		bucketManager:      bucketManager,
		archiveLister:      archiveLister,
		archiveLocker:      archiveLocker,
		multipartUploader:  multipartUploader,
		multipartInspector: multipartInspector,
		multipartRemover:   multipartRemover,
		rootAPIPath:        rootAPIPath,
		chunkSize:          chunkSize,
	}
}

func (uc *GetExternalSolutionArchive) Execute(
	ctx context.Context,
	solutionArchive *domain.SolutionArchive,
	downloadURL string,
) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Str("download_url", downloadURL).
		Msg("Getting external solution archive")

	// To avoid simultaneous downloads and deletions of the same solution archive
	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		return nil
	}

	// Get description of the solution archive
	solutionArchiveSize, err := uc.externalDownloader.GetDescription(ctx, downloadURL)
	if err != nil {
		return errors.Stamp(err)
	}

	// Iterate over the chunks
	numChunks := (solutionArchiveSize + uc.chunkSize - 1) / uc.chunkSize
	partSolutionArchive := &domain.SolutionArchive{
		Name:    solutionArchive.Name,
		Version: solutionArchive.Version,
		Hash:    solutionArchive.Hash,
		Size:    solutionArchiveSize,
	}
	for chunkIndex := range numChunks {
		start := chunkIndex * uc.chunkSize
		end := min(start+uc.chunkSize, solutionArchiveSize) - 1
		part := &domain.Part{
			SolutionArchive: partSolutionArchive,
			Meta: &domain.PartMeta{
				Start: start,
				End:   end,
			},
		}

		body, err := uc.externalDownloader.Download(ctx, downloadURL, solutionArchive.Hash, start, end, solutionArchiveSize)
		if err != nil {
			return errors.Stamp(err)
		}

		// Store the chunk in the file (streams via io.Copy); then close the HTTP body this iteration.
		part.Content = body

		_, err = uc.storePart(part)
		_ = body.Close() // nolint: errcheck // Return path uses storePart err; Close releases the connection.
		if err != nil {
			return errors.Stamp(err)
		}
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("External solution archive retrieved")

	return nil
}

func (uc *GetExternalSolutionArchive) storePart(part *domain.Part) (*domain.SolutionArchiveStatus, error) {
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
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	// Test if a session related to the solution archive from the part exists
	if solutionArchiveFromManifest.Name != part.SolutionArchive.Name ||
		solutionArchiveFromManifest.Version != part.SolutionArchive.Version ||
		solutionArchiveFromManifest.Hash != part.SolutionArchive.Hash {
		return nil, errors.From(domain.ErrPartUploaderNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found in the current session manifest").
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			WithProperty("component", part.SolutionArchive.Name).
			WithProperty("version", part.SolutionArchive.Version).
			WithProperty("hash", part.SolutionArchive.Hash).
			Throw()
	}

	// When this is the first upload for a solution archive, the size is not set in the manifest
	// so we use the size from the part
	if solutionArchiveFromManifest.Size == 0 {
		solutionArchiveFromManifest.Size = part.SolutionArchive.Size

		err = uc.multipartRemover.DeleteMultipartFile(sessionBucket, solutionArchiveFromManifest)
		if err != nil {
			return nil, errors.Intercept(err).
				WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
				Throw()
		}

		_, err = uc.multipartUploader.CreateMultipartFiles(sessionBucket, solutionArchiveFromManifest)
		if err != nil {
			return nil, errors.Intercept(err).
				WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
				Throw()
		}
	}

	part.SolutionArchive = solutionArchiveFromManifest

	solutionArchiveStatus, err := uc.multipartUploader.WritePartToMultipartFile(sessionBucket, part)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	if !solutionArchiveStatus.IsComplete() {
		return solutionArchiveStatus, nil
	}

	// Solution archive upload is complete, so let's consolidate it,
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

		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(part.SolutionArchive)
	err = uc.multipartUploader.MoveFileToRoot(sessionBucket, part.SolutionArchive.Name, solutionArchiveFileName)
	if err != nil {
		cleanUpCorrupted()

		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}
	err = uc.bucketManager.DeleteBucket(sessionBucket)
	if err != nil {
		return nil, errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, part.SolutionArchive.Name)).
			Throw()
	}

	return solutionArchiveStatus, nil
}
