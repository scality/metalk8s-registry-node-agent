package multipartuploader

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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

var _ service.MultipartUploader = &FileSystem{}

type FileSystem struct {
	logger                   *zerolog.Logger
	solutionArchivesLocation string
}

func NewFileSystem(
	logger *zerolog.Logger,
	solutionArchivesLocation string,
) *FileSystem {
	l := logger.With().
		Str("infrastructure", "multipart_uploader").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
	}
}

func (f *FileSystem) CreateMultipartFiles(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	if err := library.EnforceNamingConventions(solutionArchiveMeta.Name); err != nil {
		return nil, errors.Stamp(err)
	}

	if err := library.CheckDir(f.genBucketPath(bucketName)); err != nil {
		return nil, errors.Stamp(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		solutionArchiveMeta.Name,
	)

	err := library.CheckFile(metaFilePath)
	if err == nil {
		solutionArchiveStatus, err := f.getSolutionArchiveStatus(bucketName, solutionArchiveMeta)
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

	metaContentBytes, err := json.Marshal(solutionArchiveMeta)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to save the solution archive metadata").
			WithProperty("solution_archive", solutionArchiveMeta.Name).
			WithProperty("version", solutionArchiveMeta.Version).
			WithProperty("while", "marshalling the metadata to json format").
			Throw()
	}

	cleanUp := func() {
		os.Remove(metaFilePath)      // nolint: errcheck
		os.Remove(partsFilePath)     // nolint: errcheck
		os.Remove(recipientFilePath) // nolint: errcheck
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
		solutionArchiveMeta.Size,
		library.FileSystemDefaultFileMode,
	); err != nil {
		cleanUp()
		return nil, errors.Stamp(err)
	}

	return &domain.SolutionArchiveStatus{
		SolutionArchive: solutionArchiveMeta,
		Parts:           make(map[int64]*domain.PartMeta),
	}, nil
}

func (f *FileSystem) WritePartToMultipartFile(
	bucketName string,
	part *domain.Part,
) (*domain.SolutionArchiveStatus, error) {
	solutionArchiveStatus, err := f.getSolutionArchiveStatus(bucketName, part.SolutionArchive)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	partsFilePath := f.genMultipartPartsFilePath(bucketName, part.SolutionArchive.Name)
	recipientFilePath := f.genMultipartRecipientFilePath(bucketName, part.SolutionArchive.Name)

	partsFile, err := os.OpenFile(partsFilePath, os.O_WRONLY, library.FileSystemDefaultFileMode)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to open the parts file").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	defer partsFile.Close() // nolint: errcheck

	recipientFile, err := os.OpenFile(recipientFilePath, os.O_RDWR, library.FileSystemDefaultFileMode)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to open the recipient file").
			WithProperty("file_path", recipientFilePath).
			CausedBy(err).
			Throw()
	}

	defer recipientFile.Close() // nolint: errcheck

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

	_, err = partsFile.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to move to the end of the parts file").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	err = binary.Write(partsFile, binary.LittleEndian, part.Meta)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to write the part metadata to the parts file").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	solutionArchiveStatus.Parts[part.Meta.Start] = part.Meta

	return solutionArchiveStatus, nil
}

func (f *FileSystem) ConsolidateMultipartFile(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
	perm os.FileMode,
) error {
	solutionArchiveStatus, err := f.getSolutionArchiveStatus(bucketName, solutionArchiveMeta)
	if err != nil {
		return err
	}

	if !solutionArchiveStatus.IsComplete() {
		return errors.From(domain.ErrStorageProviderBusinessRuleViolation).
			WithIdentifier(422001).
			WithDetail("unable to consolidate multipart file because it is not complete").
			WithProperty("bucket_name", bucketName).
			WithProperty("solution_archive_name", solutionArchiveMeta.Name).
			Throw()
	}

	baseFilePath := f.genBaseMultipartFilePath(bucketName, solutionArchiveMeta.Name)
	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		solutionArchiveMeta.Name,
	)

	recipientFile, err := library.GetFile(recipientFilePath)
	if err != nil {
		return errors.Stamp(err)
	}

	defer recipientFile.Close() // nolint: errcheck

	hasher := sha256.New()
	if _, err := io.Copy(hasher, recipientFile); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to calculate the hash of the recipient file").
			WithProperty("file_path", recipientFilePath).
			CausedBy(err).
			Throw()
	}

	calculedHash := hex.EncodeToString(hasher.Sum(nil))

	if calculedHash != solutionArchiveMeta.Hash {
		return errors.From(domain.ErrStorageProviderBusinessRuleViolation).
			WithIdentifier(422001).
			WithDetail("the hash of the recipient file does not match the solution archive metadata").
			WithProperty("component", solutionArchiveMeta.Name).
			WithProperty("version", solutionArchiveMeta.Version).
			WithProperty("expected_hash", solutionArchiveMeta.Hash).
			WithProperty("calculed_hash", calculedHash).
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

func (f *FileSystem) MoveFileToRoot(
	bucketName, fileName, newFileName string,
) error {
	if err := library.EnforceNamingConventions(newFileName); err != nil {
		return errors.Stamp(err)
	}

	bucketPath := f.genBucketPath(bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return errors.Stamp(err)
	}

	filePath := f.genFileOnBucketPath(bucketName, fileName)
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

// --- shared helpers (duplicated with multipartinspector) ---

func (f *FileSystem) genBucketPath(bucketName string) string {
	return filepath.Join(f.solutionArchivesLocation, library.FileSystemBucketPrefix+bucketName)
}

func (f *FileSystem) genFileOnBucketPath(bucketName, fileName string) string {
	return filepath.Join(f.genBucketPath(bucketName), fileName)
}

func (f *FileSystem) genBaseMultipartFilePath(bucketName, fileName string) string {
	return filepath.Join(f.genBucketPath(bucketName), fileName)
}

func (f *FileSystem) genMultipartMetaFilePath(bucketName, fileName string) string {
	return f.genBaseMultipartFilePath(bucketName, fileName) + library.FileSystemMultipartMetaSuffix
}

func (f *FileSystem) genMultipartPartsFilePath(bucketName, fileName string) string {
	return f.genBaseMultipartFilePath(bucketName, fileName) + library.FileSystemMultipartPartsSuffix
}

func (f *FileSystem) genMultipartRecipientFilePath(bucketName, fileName string) string {
	return f.genBaseMultipartFilePath(bucketName, fileName) +
		library.FileSystemMultipartRecipientSuffix
}

func (f *FileSystem) genMultipartFilePaths(
	bucketName, fileName string,
) (metaFilePath, partsFilePath, recipientFilePath string) {
	return f.genMultipartMetaFilePath(bucketName, fileName),
		f.genMultipartPartsFilePath(bucketName, fileName),
		f.genMultipartRecipientFilePath(bucketName, fileName)
}

func loadSolutionArchiveMeta(meta *domain.SolutionArchive, filePath string) error {
	metaFile, err := library.GetFile(filePath)
	if err != nil {
		return errors.Stamp(err)
	}

	defer metaFile.Close() // nolint: errcheck

	if err := json.NewDecoder(metaFile).Decode(meta); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to load the metadata of the multipart file").
			WithProperty("file_path", filePath).
			WithProperty("while", "decoding the metadata from json").
			CausedBy(err).
			Throw()
	}

	return nil
}

func (f *FileSystem) getSolutionArchiveStatus(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	if err := library.CheckDir(f.genBucketPath(bucketName)); err != nil {
		return nil, errors.Stamp(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		solutionArchiveMeta.Name,
	)

	if err := library.CheckFile(metaFilePath); err != nil {
		return nil, errors.Stamp(err)
	}

	if err := library.CheckFile(partsFilePath); err != nil {
		return nil, errors.Stamp(err)
	}

	if err := library.CheckFile(recipientFilePath); err != nil {
		return nil, errors.Stamp(err)
	}

	metaFile, err := library.GetFile(metaFilePath)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	defer metaFile.Close() // nolint: errcheck

	partsFile, err := library.GetFile(partsFilePath)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	defer partsFile.Close() // nolint: errcheck

	var storedSolutionArchiveMeta domain.SolutionArchive
	if err := json.NewDecoder(metaFile).Decode(&storedSolutionArchiveMeta); err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to load the metadata of the multipart file").
			WithProperty("file_path", metaFilePath).
			WithProperty("while", "decoding the metadata from json").
			CausedBy(err).
			Throw()
	}

	if err := library.CompareSolutionArchiveMetas(solutionArchiveMeta, &storedSolutionArchiveMeta); err != nil {
		return nil, errors.Stamp(err)
	}

	partMetas := make(map[int64]*domain.PartMeta)
	for {
		var partMeta domain.PartMeta

		err = binary.Read(partsFile, binary.LittleEndian, &partMeta)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, errors.From(domain.ErrStorageProviderInternal).
				WithIdentifier(500000).
				WithDetail("unable to load the parts metadata of the multipart file").
				WithProperty("file_path", partsFilePath).
				WithProperty("while", "decoding the parts metadata from binary").
				CausedBy(err).
				Throw()
		}

		partMetas[partMeta.Start] = &partMeta
	}

	return &domain.SolutionArchiveStatus{
		SolutionArchive: solutionArchiveMeta,
		Parts:           partMetas,
	}, nil
}
