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

type DownloadPart struct {
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

func NewDownloadPart(
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
) *DownloadPart {
	l := logger.With().Str("use_case", "download_part").Logger()

	return &DownloadPart{
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

func (uc *DownloadPart) Execute(
	ctx context.Context,
	solutionArchive *domain.SolutionArchive,
	downloadURL string,
) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Str("download_url", downloadURL).
		Msg("Downloading part")

	// To avoid simultaneous downloads and deletions of the same solution archive
	uc.bucketLocker.Lock(solutionArchive)
	defer uc.bucketLocker.Unlock(solutionArchive)
	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.archiveLister.ListFiles()
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(204),
			errors.WithDetail("error on listing solution archives"),
			errors.WithProperty("usecase", "download_part"),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchive.Name,
				solutionArchive.Version,
			)),
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
			errors.WithProperty("usecase", "download_part"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchive.Version),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchive.Name,
				solutionArchive.Version,
			)),
		)
	}

	// List all the buckets
	buckets, err := uc.bucketManager.ListBuckets()
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(206),
			errors.WithDetail("error on listing buckets"),
			errors.WithProperty("usecase", "download_part"),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchive.Name,
				solutionArchive.Version,
			)),
		)
	}

	// Extract the session bucket
	sessionBucket, err := library.ExtractSessionBucket(buckets, solutionArchive)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(207),
			errors.WithDetail("failed to extract the session bucket"),
			errors.WithProperty("usecase", "download_part"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchive.Version),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchive.Name,
				solutionArchive.Version,
			)),
		)
	}

	// Load the manifest from metadata file
	solutionArchiveFromManifest, err := uc.multipartInspector.GetMultipartFile(sessionBucket)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(208),
			errors.WithDetail("error on retrieving the manifest file"),
			errors.WithProperty("usecase", "download_part"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchive.Version),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchive.Name,
				solutionArchive.Version,
			)),
		)
	}

	if !uc.checkManifestFile(solutionArchiveFromManifest, solutionArchive) {
		return errors.Wrap(domain.ErrPartDownloaderNotFound,
			errors.WithIdentifier(209),
			errors.WithDetail("error on checking the manifest file"),
			errors.WithProperty("usecase", "download_part"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchive.Version),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchive.Name,
				solutionArchive.Version,
			)),
		)
	}

	// Load the manifest from the session bucket
	solutionArchiveStatus, err := uc.multipartInspector.GetMultipartFileStatus(sessionBucket, solutionArchive)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(210),
			errors.WithDetail("error on getting the solution archive status"),
			errors.WithProperty("usecase", "download_part"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchive.Version),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchive.Name,
				solutionArchive.Version,
			)),
		)
	}

	// Iterate over the chunks
	numChunks := (solutionArchiveSize + uc.chunkSize - 1) / uc.chunkSize
	partSolutionArchive := &domain.SolutionArchive{
		Name:    solutionArchive.Name,
		Version: solutionArchive.Version,
		Size:    solutionArchiveSize,
	}
	if solutionArchive.Hash != nil {
		partSolutionArchive.Hash = solutionArchive.Hash
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
				errors.WithProperty("usecase", "download_part"),
				errors.WithProperty("solution_archive_name", solutionArchive.Name),
				errors.WithProperty("solution_archive_version", solutionArchive.Version),
				errors.WithProperty("range_start", start),
				errors.WithProperty("range_end", end),
				errors.WithProperty("instance", fmt.Sprintf(
					"%s/downloads/%s/%s",
					uc.rootAPIPath,
					solutionArchive.Name,
					solutionArchive.Version,
				)),
			)
		}

		part.Content = body

		solutionArchiveStatus, err = uc.multipartUploader.StorePart(
			sessionBucket,
			solutionArchiveFromManifest,
			part,
		)
		if err != nil {
			_ = body.Close() // nolint: errcheck // Return path uses WritePartToRecipientFile err; Close releases the connection.
			return errors.Wrap(err,
				errors.WithIdentifier(212),
				errors.WithDetail("error on storing a solution archive chunk"),
				errors.WithProperty("usecase", "download_part"),
				errors.WithProperty("solution_archive_name", solutionArchive.Name),
				errors.WithProperty("solution_archive_version", solutionArchive.Version),
				errors.WithProperty("range_start", start),
				errors.WithProperty("range_end", end),
				errors.WithProperty("instance", fmt.Sprintf(
					"%s/downloads/%s/%s",
					uc.rootAPIPath,
					solutionArchive.Name,
					solutionArchive.Version,
				)),
			)
		}

		// Close verifies the Content-Digest trailer against the computed hash.
		if err := body.Close(); err != nil {
			return errors.Wrap(err,
				errors.WithIdentifier(213),
				errors.WithDetail("error on verifying the Content-Digest trailer of the solution archive chunk"),
				errors.WithProperty("usecase", "download_part"),
				errors.WithProperty("solution_archive_name", solutionArchive.Name),
				errors.WithProperty("solution_archive_version", solutionArchive.Version),
				errors.WithProperty("range_start", start),
				errors.WithProperty("range_end", end),
				errors.WithProperty("instance", fmt.Sprintf(
					"%s/downloads/%s/%s",
					uc.rootAPIPath,
					solutionArchive.Name,
					solutionArchive.Version,
				)),
			)
		}

		if err := uc.multipartUploader.CommitPart(sessionBucket, part); err != nil {
			return errors.Wrap(err,
				errors.WithIdentifier(214),
				errors.WithDetail("error on committing a solution archive chunk"),
				errors.WithProperty("usecase", "download_part"),
				errors.WithProperty("solution_archive_name", solutionArchive.Name),
				errors.WithProperty("solution_archive_version", solutionArchive.Version),
				errors.WithProperty("range_start", start),
				errors.WithProperty("range_end", end),
				errors.WithProperty("instance", fmt.Sprintf(
					"%s/downloads/%s/%s",
					uc.rootAPIPath,
					solutionArchive.Name,
					solutionArchive.Version,
				)),
			)
		}
	}

	if !solutionArchiveStatus.IsComplete() {
		return errors.Wrap(domain.ErrPartDownloaderNotComplete,
			errors.WithIdentifier(215),
			errors.WithDetail("unable to download part because it is not complete"),
			errors.WithProperty("usecase", "download_part"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchive.Version),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchive.Name,
				solutionArchive.Version,
			)),
		)
	}

	// Solution archive is complete, so let's consolidate it,
	// move it to the storage root location and then
	// remove the bucket.
	err = uc.multipartUploader.Consolidate(
		sessionBucket,
		solutionArchiveFromManifest,
		library.FileSystemDefaultFileMode,
	)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(216),
			errors.WithDetail("error on consolidating the solution archive"),
			errors.WithProperty("usecase", "download_part"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
			errors.WithProperty("solution_archive_version", solutionArchive.Version),
			errors.WithProperty("instance", fmt.Sprintf(
				"%s/downloads/%s/%s",
				uc.rootAPIPath,
				solutionArchive.Name,
				solutionArchive.Version,
			)),
		)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Part downloaded")

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
