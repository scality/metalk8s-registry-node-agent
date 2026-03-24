package usecase

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type GetExternalSolutionArchive struct {
	logger             *zerolog.Logger
	locker             sync.Locker
	fileLister         service.FileLister
	bucketManager      service.BucketManager
	multipartUploader  service.MultipartUploader
	multipartInspector service.MultipartInspector
	externalDownloader service.ExternalDownloader
	rootAPIPath        string
	chunkSize          int64
}

func NewGetExternalSolutionArchive(
	locker sync.Locker,
	fileLister service.FileLister,
	bucketManager service.BucketManager,
	multipartUploader service.MultipartUploader,
	multipartInspector service.MultipartInspector,
	logger *zerolog.Logger,
	externalDownloader service.ExternalDownloader,
	rootAPIPath string,
	chunkSize int64,
) *GetExternalSolutionArchive {
	l := logger.With().Str("use_case", "get_external_solution_archive").Logger()
	return &GetExternalSolutionArchive{
		logger:             &l,
		locker:             locker,
		fileLister:         fileLister,
		bucketManager:      bucketManager,
		multipartUploader:  multipartUploader,
		multipartInspector: multipartInspector,
		externalDownloader: externalDownloader,
		rootAPIPath:        rootAPIPath,
		chunkSize:          chunkSize,
	}
}

func (uc *GetExternalSolutionArchive) Execute(solutionArchive *domain.SolutionArchive, downloadURL string) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Str("download_url", downloadURL).
		Msg("Getting external solution archive")

	uc.locker.Lock()
	defer uc.locker.Unlock()

	// List all solution archives in the storage
	// matching solutionArchiveStorageNamePattern
	fileNames, err := uc.fileLister.ListFiles()
	if err != nil {
		return errors.Stamp(err)
	}

	// Check if the solution archive exists in the storage
	if library.SolutionArchiveExists(solutionArchive, fileNames) {
		return nil
	}

	// Get description of the solution archive
	solutionArchiveSize, err := uc.externalDownloader.GetDescription(downloadURL)
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

		body, err := uc.externalDownloader.Download(downloadURL, solutionArchive.Hash, start, end, solutionArchiveSize)
		if err != nil {
			return errors.Stamp(err)
		}

		// Store the chunk in the file (streams via io.Copy); then close the HTTP body this iteration.
		part.Content = body
		solutionArchiveStatus, err = uc.storePart(sessionBucket, solutionArchiveFromManifest, part)
		_ = body.Close() // nolint: errcheck // Return path uses storePart err; Close releases the connection.
		if err != nil {
			return errors.Stamp(err)
		}
	}

	return nil
}

func (uc *GetExternalSolutionArchive) checkManifestFile(solutionArchiveFromManifest, solutionArchive *domain.SolutionArchive) error {
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

func (uc *GetExternalSolutionArchive) storePart(
	sessionBucket string,
	solutionArchiveFromManifest *domain.SolutionArchive,
	part *domain.Part,
) (*domain.SolutionArchiveStatus, error) {
	// When this is the first upload for a solution archive, the size is not set in the manifest
	// so we use the size from the part
	if solutionArchiveFromManifest.Size == 0 {
		solutionArchiveFromManifest.Size = part.SolutionArchive.Size

		err := uc.multipartInspector.DeleteMultipartFile(sessionBucket, solutionArchiveFromManifest)
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
		if err := uc.multipartInspector.DeleteMultipartFile(sessionBucket, part.SolutionArchive); err != nil {
			uc.logger.Error().Err(err).Any("solution archive", part.SolutionArchive).Msg("failed to delete multipart file")
		}

		if _, err := uc.multipartUploader.CreateMultipartFiles(sessionBucket, part.SolutionArchive); err != nil {
			uc.logger.Error().Err(err).Any("solution archive", part.SolutionArchive).Msg("failed to create multipart file")
		}
	}

	err = uc.multipartUploader.ConsolidateMultipartFile(sessionBucket, part.SolutionArchive, library.FileSystemDefaultFileMode)
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
