package multipartuploader

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type FileSystem struct {
	logger                   *zerolog.Logger
	solutionArchivesLocation string
	multipartInspector       service.MultipartInspector
	multipartRemover         service.MultipartRemover
	bucketManager            service.BucketManager
}

func NewFileSystem(
	logger *zerolog.Logger,
	solutionArchivesLocation string,
	multipartInspector service.MultipartInspector,
	multipartRemover service.MultipartRemover,
	bucketManager service.BucketManager,
) service.MultipartUploader {
	l := logger.With().
		Str("infrastructure", "multipart_uploader").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
		multipartInspector:       multipartInspector,
		multipartRemover:         multipartRemover,
		bucketManager:            bucketManager,
	}
}

var _ service.MultipartUploader = &FileSystem{}

// CreateMultipartFiles creates into a bucket: a metadata file, a multipart file recipient and a parts synthesis file.
func (f *FileSystem) CreateMultipartFiles(
	bucketName string,
	solutionArchive *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	if err := library.EnforceNamingConventions(solutionArchive.Name); err != nil {
		return nil, errors.Stamp(err)
	}

	if err := library.CheckDir(library.GenBucketPath(f.solutionArchivesLocation, bucketName)); err != nil {
		return nil, errors.Stamp(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := library.GenMultipartFilePaths(
		f.solutionArchivesLocation,
		bucketName,
		solutionArchive.Name,
	)

	err := library.CheckFile(metaFilePath)
	if err == nil {
		solutionArchiveStatus, err := f.multipartInspector.GetMultipartFileStatus(bucketName, solutionArchive)
		if err != nil {
			return nil, errors.Stamp(err)
		}

		return solutionArchiveStatus, nil
	}

	if !errors.Is(err,
		errors.Intercept(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			Throw()) {
		return nil, errors.Stamp(err)
	}

	metaContentBytes, err := json.Marshal(solutionArchive)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to save the solution archive metadata").
			WithProperty("solution_archive", solutionArchive.Name).
			WithProperty("version", solutionArchive.Version).
			WithProperty("while", "marshalling the metadata to json format").
			Throw()
	}

	cleanUp := func() {
		os.Remove(metaFilePath)      // nolint: errcheck // No *PathError error possible.
		os.Remove(partsFilePath)     // nolint: errcheck // No *PathError error possible.
		os.Remove(recipientFilePath) // nolint: errcheck // No *PathError error possible.
	}

	if err := library.SaveFile(
		metaFilePath,
		bytes.NewReader(metaContentBytes),
		library.FileSystemDefaultFileMode,
	); err != nil {
		cleanUp()

		return nil, errors.Stamp(err)
	}

	if err := library.CreateEmptyFile(
		partsFilePath,
		0,
		library.FileSystemDefaultFileMode,
	); err != nil {
		cleanUp()

		return nil, errors.Stamp(err)
	}

	if err := library.CreateEmptyFile(
		recipientFilePath,
		solutionArchive.Size,
		library.FileSystemDefaultFileMode,
	); err != nil {
		cleanUp()

		return nil, errors.Stamp(err)
	}

	return &domain.SolutionArchiveStatus{
		SolutionArchive: solutionArchive,
		Parts:           make(map[int64]*domain.PartMeta),
	}, nil
}

// StorePart writes the content of the given part into the multipart file recipient
// on the bucket indicated by the given bucketName.
func (f *FileSystem) StorePart(
	bucketName string,
	solutionArchiveFromManifest *domain.SolutionArchive,
	part *domain.Part,
) (*domain.SolutionArchiveStatus, error) {
	// When this is the first stored part for a solution archive, the size is not set in the manifest
	// so we use the size from the part. We need to recreate the multipart files with the correct size.
	if solutionArchiveFromManifest.Size == 0 {
		solutionArchiveFromManifest.Size = part.SolutionArchive.Size

		if err := f.multipartRemover.DeleteMultipartFile(bucketName, solutionArchiveFromManifest); err != nil {
			return nil, errors.Stamp(err)
		}

		if _, err := f.CreateMultipartFiles(bucketName, solutionArchiveFromManifest); err != nil {
			return nil, errors.Stamp(err)
		}
	}
	part.SolutionArchive = solutionArchiveFromManifest

	solutionArchiveStatus, err := f.multipartInspector.GetMultipartFileStatus(bucketName, part.SolutionArchive)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	recipientFilePath := library.GenMultipartRecipientFilePath(
		f.solutionArchivesLocation,
		bucketName,
		part.SolutionArchive.Name,
	)

	recipientFile, err := os.OpenFile(recipientFilePath, os.O_RDWR, library.FileSystemDefaultFileMode)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to open the recipient file").
			WithProperty("file_path", recipientFilePath).
			CausedBy(err).
			Throw()
	}

	defer recipientFile.Close() // nolint: errcheck // No error check on defer.

	_, err = recipientFile.Seek(part.Meta.Start, io.SeekStart)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to move to the start of the part in the recipient file").
			WithProperty("file_path", recipientFilePath).
			WithProperty("part_start", part.Meta.Start).
			CausedBy(err).
			Throw()
	}

	written, err := io.Copy(recipientFile, part.Content)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to write the part to the recipient file").
			WithProperty("file_path", recipientFilePath).
			WithProperty("part_size", part.Meta.Size()).
			CausedBy(err).
			Throw()
	}

	if written != part.Meta.Size() {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("the part was not fully written to the recipient file").
			WithProperty("file_path", recipientFilePath).
			WithProperty("part_size", part.Meta.Size()).
			WithProperty("written", written).
			Throw()
	}

	solutionArchiveStatus.Parts[part.Meta.Start] = part.Meta

	return solutionArchiveStatus, nil
}

// CommitPart properly updates the .part files
// with the metadata of the given part.
func (f *FileSystem) CommitPart(bucketName string, part *domain.Part) error {
	partsFilePath := library.GenMultipartPartsFilePath(
		f.solutionArchivesLocation,
		bucketName,
		part.SolutionArchive.Name,
	)

	partsFile, err := os.OpenFile(partsFilePath, os.O_WRONLY, library.FileSystemDefaultFileMode)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to open the parts file").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	defer partsFile.Close() // nolint: errcheck // No error check on defer.

	_, err = partsFile.Seek(0, io.SeekEnd)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to move to the end of the parts file").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	err = binary.Write(partsFile, binary.LittleEndian, part.Meta)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to write the part metadata to the parts file").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	return nil
}

// consolidateMultipartFile consolidates all the parts of a multipart file in a single flat file
// into the same bucket it is located.
func (f *FileSystem) consolidateMultipartFile(
	bucketName string,
	solutionArchive *domain.SolutionArchive,
	perm os.FileMode,
) error {
	solutionArchiveStatus, err := f.multipartInspector.GetMultipartFileStatus(bucketName, solutionArchive)
	if err != nil {
		return errors.Stamp(err)
	}

	if !solutionArchiveStatus.IsComplete() {
		return errors.From(domain.ErrStorageProviderBusinessRuleViolation).
			WithIdentifier(422001).
			WithDetail("unable to consolidate multipart file because it is not complete").
			WithProperty("bucket_name", bucketName).
			WithProperty("solution_archive_name", solutionArchive.Name).
			Throw()
	}

	baseFilePath := library.GenBaseMultipartFilePath(f.solutionArchivesLocation, bucketName, solutionArchive.Name)
	metaFilePath, partsFilePath, recipientFilePath := library.GenMultipartFilePaths(
		f.solutionArchivesLocation,
		bucketName,
		solutionArchive.Name,
	)

	// Calculate the SHA256 hash of the recipient file
	calculatedHash, err := library.HashFile(recipientFilePath)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to calculate the hash of the recipient file").
			WithProperty("file_path", recipientFilePath).
			CausedBy(err).
			Throw()
	}

	if calculatedHash != solutionArchive.Hash {
		return errors.From(domain.ErrStorageProviderBusinessRuleViolation).
			WithIdentifier(422001).
			WithDetail("the hash of the recipient file does not match the solution archive metadata").
			WithProperty("component", solutionArchive.Name).
			WithProperty("version", solutionArchive.Version).
			WithProperty("expected_hash", solutionArchive.Hash).
			WithProperty("calculated_hash", calculatedHash).
			Throw()
	}

	if err := os.Rename(recipientFilePath, baseFilePath); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to rename the recipient file").
			WithProperty("from", recipientFilePath).
			WithProperty("to", baseFilePath).
			CausedBy(err).
			Throw()
	}

	if err := os.Chmod(baseFilePath, perm); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to change the permissions of the recipient file").
			WithProperty("file_path", baseFilePath).
			WithProperty("permissions", perm).
			CausedBy(err).
			Throw()
	}

	// Remove other files
	problems := make(map[string]any)

	if err := os.Remove(metaFilePath); err != nil {
		problems["problem_remove_meta_file"] = err
	}

	if err := os.Remove(partsFilePath); err != nil {
		problems["problem_remove_parts_file"] = err
	}

	if len(problems) > 0 {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to clean up multipart file bundle").
			WithProperties(problems).
			Throw()
	}

	return nil
}

// moveFileToRoot moves a file from a bucket to the root location in the storage.
func (f *FileSystem) moveFileToRoot(
	bucketName, fileName, newFileName string,
) error {
	if err := library.EnforceNamingConventions(newFileName); err != nil {
		return errors.Stamp(err)
	}

	bucketPath := library.GenBucketPath(f.solutionArchivesLocation, bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return errors.Stamp(err)
	}

	filePath := library.GenBaseMultipartFilePath(f.solutionArchivesLocation, bucketName, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return errors.Stamp(err)
	}

	newFilePath := filepath.Join(f.solutionArchivesLocation, newFileName)
	if err := os.Remove(newFilePath); err != nil && !os.IsNotExist(err) {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unexpected error while moving the file to the root location").
			WithProperty("current_file_path", filePath).
			WithProperty("new_file_path", newFilePath).
			WithProperty("while", "removing existing file from the root location").
			CausedBy(err).
			Throw()
	}

	if err := os.Rename(filePath, newFilePath); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unexpected error while moving the file to the root location").
			WithProperty("current_file_path", filePath).
			WithProperty("new_file_path", newFilePath).
			WithProperty("while", "moving the file from bucket to root location").
			CausedBy(err).
			Throw()
	}

	return nil
}

// Consolidate consolidates all the parts of a multipart file
// in a single flat file into the same bucket it is located and moves it to the root location.
func (f *FileSystem) Consolidate(
	bucketName string,
	solutionArchive *domain.SolutionArchive,
	perm os.FileMode,
) error {
	cleanUpCorrupted := func() {
		if err := f.multipartRemover.DeleteMultipartFile(bucketName, solutionArchive); err != nil {
			f.logger.Error().Err(err).
				Any("solution archive", solutionArchive).
				Msg("failed to delete multipart file")
		}

		if _, err := f.CreateMultipartFiles(bucketName, solutionArchive); err != nil {
			f.logger.Error().Err(err).
				Any("solution archive", solutionArchive).
				Msg("failed to create multipart file")
		}
	}

	err := f.consolidateMultipartFile(
		bucketName,
		solutionArchive,
		perm,
	)
	if err != nil {
		cleanUpCorrupted()
		return errors.Stamp(err)
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(solutionArchive)
	err = f.moveFileToRoot(bucketName, solutionArchive.Name, solutionArchiveFileName)
	if err != nil {
		cleanUpCorrupted()
		return errors.Stamp(err)
	}
	err = f.bucketManager.DeleteBucket(bucketName)
	if err != nil {
		return errors.Stamp(err)
	}

	return nil
}
