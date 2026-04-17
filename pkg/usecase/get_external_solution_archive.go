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
	bucketLocker       service.LockerUnlocker
	multipartUploader  service.MultipartUploader
	multipartInspector service.MultipartInspector
	rootAPIPath        string
	chunkSize          int64
}

func NewGetExternalSolutionArchive(
	logger *zerolog.Logger,
	externalDownloader service.ExternalDownloader,
	bucketManager service.BucketManager,
	archiveLister service.ArchiveLister,
	archiveLocker service.LockerUnlocker,
	bucketLocker service.LockerUnlocker,
	multipartUploader service.MultipartUploader,
	multipartInspector service.MultipartInspector,
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
		bucketLocker:       bucketLocker,
		multipartUploader:  multipartUploader,
		multipartInspector: multipartInspector,
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
	uc.bucketLocker.Lock(solutionArchive)
	defer uc.bucketLocker.Unlock(solutionArchive)
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

	// List all the buckets
	buckets, err := uc.bucketManager.ListBuckets()
	if err != nil {
		return errors.Stamp(err)
	}

	// Extract the session bucket
	sessionBucket, err := library.ExtractSessionBucket(buckets, solutionArchive)
	if err != nil {
		return errors.Intercept(err).
			WithDetail("failed to extract the session bucket").
			Throw()
	}

	// Load the manifest from metadata file
	solutionArchiveFromManifest, err := uc.multipartInspector.GetMultipartFile(sessionBucket)
	if err != nil {
		return errors.Intercept(err).
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, solutionArchive.Name)).
			Throw()
	}

	err = uc.checkManifestFile(solutionArchiveFromManifest, solutionArchive)
	if err != nil {
		return errors.Stamp(err)
	}

	// Load the manifest from the session bucket
	solutionArchiveStatus, err := uc.multipartInspector.GetMultipartFileStatus(sessionBucket, solutionArchive)
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

		// Check if the part has already been downloaded
		if solutionArchiveStatus.ContainsPart(part.Meta) {
			continue
		}

		body, err := uc.externalDownloader.Download(ctx, downloadURL, solutionArchive.Hash, start, end, solutionArchiveSize)
		if err != nil {
			return errors.Stamp(err)
		}

		// Store the chunk in the file (streams via io.Copy); then close the HTTP body this iteration.
		part.Content = body
		solutionArchiveStatus, err = uc.multipartUploader.StorePart(sessionBucket, solutionArchiveFromManifest, part)
		_ = body.Close() // nolint: errcheck // Return path uses StorePart err; Close releases the connection.
		if err != nil {
			return errors.Intercept(err).
				WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, solutionArchive.Name)).
				Throw()
		}
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("External solution archive retrieved")

	return nil
}

func (uc *GetExternalSolutionArchive) checkManifestFile(
	solutionArchiveFromManifest *domain.SolutionArchive,
	solutionArchive *domain.SolutionArchive,
) error {
	// Test if a session related to the solution archive from the part exists
	if solutionArchiveFromManifest.Name != solutionArchive.Name ||
		solutionArchiveFromManifest.Version != solutionArchive.Version ||
		solutionArchiveFromManifest.Hash != solutionArchive.Hash {
		return errors.From(domain.ErrPartUploaderNotFound).
			WithIdentifier(404000).
			WithDetail("solution archive not found in the current session manifest").
			WithProperty("instance", fmt.Sprintf("%s/downloads/%s", uc.rootAPIPath, solutionArchive.Name)).
			WithProperty("component", solutionArchive.Name).
			WithProperty("version", solutionArchive.Version).
			WithProperty("hash", solutionArchive.Hash).
			Throw()
	}
	return nil
}
