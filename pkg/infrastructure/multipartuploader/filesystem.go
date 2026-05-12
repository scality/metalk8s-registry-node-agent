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
		return nil, errors.Wrap(err)
	}

	if err := library.CheckDir(library.GenBucketPath(f.solutionArchivesLocation, bucketName)); err != nil {
		return nil, errors.Wrap(err)
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
			return nil, errors.Wrap(err)
		}

		return solutionArchiveStatus, nil
	}

	if !errors.Is(err, domain.ErrStorageProviderNotFound) {
		return nil, errors.Wrap(err)
	}

	metaContentBytes, err := json.Marshal(solutionArchive)
	if err != nil {
		return nil, errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to save the solution archive metadata"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
			errors.WithProperty("version", solutionArchive.Version),
			errors.WithProperty("while", "marshalling the metadata to json format"),
			errors.CausedBy(err),
		)
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

		return nil, errors.Wrap(err)
	}

	if err := library.CreateEmptyFile(
		partsFilePath,
		0,
		library.FileSystemDefaultFileMode,
	); err != nil {
		cleanUp()

		return nil, errors.Wrap(err)
	}

	if err := library.CreateEmptyFile(
		recipientFilePath,
		solutionArchive.Size,
		library.FileSystemDefaultFileMode,
	); err != nil {
		cleanUp()

		return nil, errors.Wrap(err)
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
			return nil, errors.Wrap(err)
		}

		if _, err := f.CreateMultipartFiles(bucketName, solutionArchiveFromManifest); err != nil {
			return nil, errors.Wrap(err)
		}
	}
	part.SolutionArchive = solutionArchiveFromManifest

	solutionArchiveStatus, err := f.multipartInspector.GetMultipartFileStatus(bucketName, part.SolutionArchive)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	recipientFilePath := library.GenMultipartRecipientFilePath(
		f.solutionArchivesLocation,
		bucketName,
		part.SolutionArchive.Name,
	)

	recipientFile, err := os.OpenFile(recipientFilePath, os.O_RDWR, library.FileSystemDefaultFileMode)
	if err != nil {
		return nil, errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to open the recipient file"),
			errors.WithProperty("file_path", recipientFilePath),
			errors.CausedBy(err),
		)
	}

	defer recipientFile.Close() // nolint: errcheck // No error check on defer.

	_, err = recipientFile.Seek(part.Meta.Start, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to move to the start of the part in the recipient file"),
			errors.WithProperty("file_path", recipientFilePath),
			errors.WithProperty("part_start", part.Meta.Start),
			errors.CausedBy(err),
		)
	}

	written, err := io.Copy(recipientFile, part.Content)
	if err != nil {
		return nil, errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to write the part to the recipient file"),
			errors.WithProperty("file_path", recipientFilePath),
			errors.WithProperty("part_size", part.Meta.Size()),
			errors.CausedBy(err),
		)
	}

	if written != part.Meta.Size() {
		return nil, errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("the part was not fully written to the recipient file"),
			errors.WithProperty("file_path", recipientFilePath),
			errors.WithProperty("part_size", part.Meta.Size()),
			errors.WithProperty("written", written),
		)
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
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to open the parts file"),
			errors.WithProperty("file_path", partsFilePath),
			errors.CausedBy(err),
		)
	}

	defer partsFile.Close() // nolint: errcheck // No error check on defer.

	_, err = partsFile.Seek(0, io.SeekEnd)
	if err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to move to the end of the parts file"),
			errors.WithProperty("file_path", partsFilePath),
			errors.CausedBy(err),
		)
	}

	err = binary.Write(partsFile, binary.LittleEndian, part.Meta)
	if err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to write the part metadata to the parts file"),
			errors.WithProperty("file_path", partsFilePath),
			errors.CausedBy(err),
		)
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
		return errors.Wrap(err)
	}

	if !solutionArchiveStatus.IsComplete() {
		return errors.Wrap(domain.ErrStorageProviderBusinessRuleViolation,
			errors.WithIdentifier(422001),
			errors.WithDetail("unable to consolidate multipart file because it is not complete"),
			errors.WithProperty("bucket_name", bucketName),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
		)
	}

	baseFilePath := library.GenBaseMultipartFilePath(f.solutionArchivesLocation, bucketName, solutionArchive.Name)
	metaFilePath, partsFilePath, recipientFilePath := library.GenMultipartFilePaths(
		f.solutionArchivesLocation,
		bucketName,
		solutionArchive.Name,
	)

	// Validate the recipient integrity, if solution archive hash is set
	if solutionArchive.Hash != nil {
		calculatedHash, err := library.HashFile(recipientFilePath)
		if err != nil {
			return errors.Wrap(domain.ErrStorageProviderInternal,
				errors.WithIdentifier(500000),
				errors.WithDetail("unable to calculate the hash of the recipient file"),
				errors.WithProperty("file_path", recipientFilePath),
				errors.CausedBy(err),
			)
		}

		if calculatedHash != *solutionArchive.Hash {
			return errors.Wrap(domain.ErrStorageProviderBusinessRuleViolation,
				errors.WithIdentifier(422001),
				errors.WithDetail("the hash of the recipient file does not match the solution archive metadata"),
				errors.WithProperty("component", solutionArchive.Name),
				errors.WithProperty("version", solutionArchive.Version),
				errors.WithProperty("expected_hash", *solutionArchive.Hash),
				errors.WithProperty("calculated_hash", calculatedHash),
			)
		}
	}

	if err := os.Rename(recipientFilePath, baseFilePath); err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to rename the recipient file"),
			errors.WithProperty("from_path", recipientFilePath),
			errors.WithProperty("to_path", baseFilePath),
			errors.CausedBy(err),
		)
	}

	if err := os.Chmod(baseFilePath, perm); err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to change the permissions of the recipient file"),
			errors.WithProperty("file_path", baseFilePath),
			errors.WithProperty("permissions", perm),
			errors.CausedBy(err),
		)
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
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to clean up multipart file bundle"),
			errors.WithProperty("problems", problems),
		)
	}

	return nil
}

// moveFileToRoot moves a file from a bucket to the root location in the storage.
func (f *FileSystem) moveFileToRoot(
	bucketName, fileName, newFileName string,
) error {
	if err := library.EnforceNamingConventions(newFileName); err != nil {
		return errors.Wrap(err)
	}

	bucketPath := library.GenBucketPath(f.solutionArchivesLocation, bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return errors.Wrap(err)
	}

	filePath := library.GenBaseMultipartFilePath(f.solutionArchivesLocation, bucketName, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return errors.Wrap(err)
	}

	newFilePath := filepath.Join(f.solutionArchivesLocation, newFileName)
	if err := os.Remove(newFilePath); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unexpected error while moving the file to the root location"),
			errors.WithProperty("current_file_path", filePath),
			errors.WithProperty("new_file_path", newFilePath),
			errors.WithProperty("while", "removing existing file from the root location"),
			errors.CausedBy(err),
		)
	}

	if err := os.Rename(filePath, newFilePath); err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unexpected error while moving the file to the root location"),
			errors.WithProperty("current_file_path", filePath),
			errors.WithProperty("new_file_path", newFilePath),
			errors.WithProperty("while", "moving the file from bucket to root location"),
			errors.CausedBy(err),
		)
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
		return errors.Wrap(err)
	}

	solutionArchiveFileName := library.GenSolutionArchiveFileName(solutionArchive)
	err = f.moveFileToRoot(bucketName, solutionArchive.Name, solutionArchiveFileName)
	if err != nil {
		cleanUpCorrupted()
		return errors.Wrap(err)
	}
	err = f.bucketManager.DeleteBucket(bucketName)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}
