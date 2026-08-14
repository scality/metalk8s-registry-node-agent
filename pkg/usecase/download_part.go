package usecase

import (
	"context"
	"log/slog"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type DownloadPart struct {
	logger             *slog.Logger
	externalDownloader service.ExternalDownloader
	bucketManager      service.BucketManager
	archiveLister      service.ArchiveLister
	archiveLocker      service.LockerUnlocker
	bucketLocker       service.LockerUnlocker
	multipartStorer    service.MultipartStorer
	multipartInspector service.MultipartInspector
	rootAPIPath        string
	chunkSize          int64
}

func NewDownloadPart(
	logger *slog.Logger,
	externalDownloader service.ExternalDownloader,
	bucketManager service.BucketManager,
	archiveLister service.ArchiveLister,
	archiveLocker service.LockerUnlocker,
	bucketLocker service.LockerUnlocker,
	multipartStorer service.MultipartStorer,
	multipartInspector service.MultipartInspector,
	rootAPIPath string,
	chunkSize int64,
) *DownloadPart {
	return &DownloadPart{
		logger:             logger.With(slog.String("use_case", "download_part")),
		externalDownloader: externalDownloader,
		bucketManager:      bucketManager,
		archiveLister:      archiveLister,
		archiveLocker:      archiveLocker,
		bucketLocker:       bucketLocker,
		multipartStorer:    multipartStorer,
		multipartInspector: multipartInspector,
		rootAPIPath:        rootAPIPath,
		chunkSize:          chunkSize,
	}
}

func (uc *DownloadPart) Execute(
	ctx context.Context,
	solutionArchive *domain.SolutionArchive,
	downloadURL string,
) error {
	uc.logger.DebugContext(ctx, "Downloading part",
		slog.Any("solution_archive", solutionArchive),
		slog.String("download_url", downloadURL),
	)

	// To avoid simultaneous downloads and deletions of the same solution archive
	uc.bucketLocker.Lock(solutionArchive)
	defer uc.bucketLocker.Unlock(solutionArchive)
	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

	properties := solutionArchive.GetErrorProperties(
		"download_part",
		solutionArchive.GetDownloadURL(uc.rootAPIPath),
	)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(204),
			errors.WithDetail("error on listing solution archives"),
			errors.WithProperties(properties),
		)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		return nil
	}

	// Get description of the solution archive
	solutionArchiveSize, err := uc.externalDownloader.GetDescription(ctx, downloadURL)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(205),
			errors.WithDetail("error on getting description of the solution archive"),
			errors.WithProperties(properties),
		)
	}

	// List all the buckets
	buckets, err := uc.bucketManager.ListBuckets()
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(206),
			errors.WithDetail("error on listing buckets"),
			errors.WithProperties(properties),
		)
	}

	// Extract the session bucket
	sessionBucket, err := library.ExtractSessionBucket(buckets, solutionArchive)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(207),
			errors.WithDetail("failed to extract the session bucket"),
			errors.WithProperties(properties),
		)
	}

	// Load the manifest from metadata file
	solutionArchiveFromManifest, err := uc.multipartInspector.GetMultipartFile(ctx, sessionBucket)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(208),
			errors.WithDetail("error on retrieving the manifest file"),
			errors.WithProperties(properties),
		)
	}

	if !uc.checkManifestFile(solutionArchiveFromManifest, solutionArchive) {
		return errors.Wrap(domain.ErrPartDownloaderNotFound,
			errors.WithIdentifier(209),
			errors.WithDetail("error on checking the manifest file"),
			errors.WithProperties(properties),
		)
	}

	// Load the manifest from the session bucket
	solutionArchiveStatus, err := uc.multipartInspector.GetMultipartFileStatus(ctx, sessionBucket, solutionArchive)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(210),
			errors.WithDetail("error on getting the solution archive status"),
			errors.WithProperties(properties),
		)
	}

	// Iterate over the chunks
	numChunks := (solutionArchiveSize + uc.chunkSize - 1) / uc.chunkSize
	partSolutionArchive := &domain.SolutionArchive{
		Name:    solutionArchive.Name,
		Version: solutionArchive.Version,
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

		body, err := uc.externalDownloader.DownloadPart(ctx, downloadURL, start, end, solutionArchiveSize)
		if err != nil {
			return errors.Wrap(err,
				errors.WithIdentifier(211),
				errors.WithDetail("error on downloading a solution archive chunk"),
				errors.WithProperties(properties),
				errors.WithProperty("range_start", start),
				errors.WithProperty("range_end", end),
			)
		}

		part.Content = body

		solutionArchiveStatus, err = uc.multipartStorer.StorePart(
			ctx,
			sessionBucket,
			solutionArchiveFromManifest,
			part,
		)
		if err != nil {
			_ = body.Close() // nolint: errcheck // Return path uses WritePartToRecipientFile err; Close releases the connection.
			return errors.Wrap(err,
				errors.WithIdentifier(212),
				errors.WithDetail("error on storing a solution archive chunk"),
				errors.WithProperties(properties),
				errors.WithProperty("range_start", start),
				errors.WithProperty("range_end", end),
			)
		}

		// Close verifies the Content-Digest trailer against the computed hash.
		if err := body.Close(); err != nil {
			return errors.Wrap(err,
				errors.WithIdentifier(213),
				errors.WithDetail("error on verifying the Content-Digest trailer of the solution archive chunk"),
				errors.WithProperties(properties),
				errors.WithProperty("range_start", start),
				errors.WithProperty("range_end", end),
			)
		}

		if err := uc.multipartStorer.CommitPart(ctx, sessionBucket, part); err != nil {
			return errors.Wrap(err,
				errors.WithIdentifier(214),
				errors.WithDetail("error on committing a solution archive chunk"),
				errors.WithProperties(properties),
				errors.WithProperty("range_start", start),
				errors.WithProperty("range_end", end),
			)
		}
	}

	if !solutionArchiveStatus.IsComplete() {
		return errors.Wrap(domain.ErrPartDownloaderNotComplete,
			errors.WithIdentifier(215),
			errors.WithDetail("unable to download part because it is not complete"),
			errors.WithProperties(properties),
		)
	}

	// Solution archive is complete, so let's consolidate it,
	// move it to the storage root location and then
	// remove the bucket.
	err = uc.multipartStorer.Consolidate(
		ctx,
		sessionBucket,
		solutionArchiveFromManifest,
		library.FileSystemDefaultFileMode,
	)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(216),
			errors.WithDetail("error on consolidating the solution archive"),
			errors.WithProperties(properties),
		)
	}

	uc.logger.DebugContext(ctx, "Part downloaded",
		slog.Any("solution_archive", solutionArchive),
	)

	return nil
}

func (uc *DownloadPart) checkManifestFile(
	solutionArchiveFromManifest *domain.SolutionArchive,
	solutionArchive *domain.SolutionArchive,
) bool {
	// Test if a session related to the solution archive from the part exists
	return solutionArchiveFromManifest.Name == solutionArchive.Name &&
		solutionArchiveFromManifest.Version == solutionArchive.Version
}
