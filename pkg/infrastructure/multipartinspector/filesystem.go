package multipartinspector

import (
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

var _ service.MultipartInspector = &FileSystem{}

type FileSystem struct {
	logger                   *zerolog.Logger
	solutionArchivesLocation string
}

func NewFileSystem(
	logger *zerolog.Logger,
	solutionArchivesLocation string,
) *FileSystem {
	l := logger.With().
		Str("infrastructure", "multipart_inspector").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
	}
}

func (f *FileSystem) GetMultipartFile(bucketName string) (*domain.SolutionArchive, error) {
	bucketPath := f.genBucketPath(bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return nil, errors.Stamp(err)
	}

	solutionArchiveMeta, err := f.getSolutionArchiveMeta(bucketPath)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return solutionArchiveMeta, nil
}

func (f *FileSystem) GetMultipartFileStatus(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	return f.getSolutionArchiveStatus(bucketName, solutionArchiveMeta)
}

func (f *FileSystem) DeleteMultipartFile(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) error {
	if err := library.CheckDir(f.genBucketPath(bucketName)); err != nil {
		return errors.Stamp(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		solutionArchiveMeta.Name,
	)

	if err := library.CheckFile(metaFilePath); err != nil {
		return errors.Stamp(err)
	}

	if err := library.CheckFile(partsFilePath); err != nil {
		return errors.Stamp(err)
	}

	if err := library.CheckFile(recipientFilePath); err != nil {
		return errors.Stamp(err)
	}

	problems := make(map[string]any)

	if err := os.Remove(metaFilePath); err != nil {
		problems["problem_remove_meta_file"] = err
	}

	if err := os.Remove(partsFilePath); err != nil {
		problems["problem_remove_parts_file"] = err
	}

	if err := os.Remove(recipientFilePath); err != nil {
		problems["problem_remove_recipient_filer"] = err
	}

	if len(problems) > 0 {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to delete the multipart file").
			WithProperties(problems).
			Throw()
	}

	return nil
}

// --- private helpers ---

func (f *FileSystem) getSolutionArchiveMeta(bucketPath string) (*domain.SolutionArchive, error) {
	metaFileNames, err := library.ListDirContentNames(bucketPath, library.MultipartMetaFilter)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	solutionArchiveMetas := make([]*domain.SolutionArchive, 0, len(metaFileNames))

	for _, metaFileName := range metaFileNames {
		var solutionArchiveMeta domain.SolutionArchive

		if err := loadSolutionArchiveMeta(&solutionArchiveMeta, filepath.Join(bucketPath, metaFileName)); err != nil {
			return nil, errors.Stamp(err)
		}

		solutionArchiveMetas = append(solutionArchiveMetas, &solutionArchiveMeta)
	}

	solutionArchiveMetas = f.filterOrphansMeta(bucketPath, solutionArchiveMetas)
	if len(solutionArchiveMetas) < 1 {
		return nil, errors.From(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			WithDetail("no multipart files found in the bucket").
			WithProperty("bucket_name", bucketPath).
			Throw()
	}
	if len(solutionArchiveMetas) > 1 {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("too many multipart files found in the bucket").
			WithProperty("bucket_name", bucketPath).
			Throw()
	}

	return solutionArchiveMetas[0], nil
}

func (f *FileSystem) filterOrphansMeta(
	bucketPath string,
	solutionArchives []*domain.SolutionArchive,
) []*domain.SolutionArchive {
	filteredSolutionArchives := make([]*domain.SolutionArchive, 0, len(solutionArchives))

	for _, solutionArchive := range solutionArchives {
		if err := library.CheckFile(
			filepath.Join(
				bucketPath,
				solutionArchive.Name+library.FileSystemMultipartPartsSuffix,
			),
		); err != nil {
			f.logger.Warn().Err(err).Msg("The parts file is missing for the multipart file.")
			continue
		}

		if err := library.CheckFile(
			filepath.Join(
				bucketPath,
				solutionArchive.Name+library.FileSystemMultipartRecipientSuffix,
			),
		); err != nil {
			f.logger.Warn().Err(err).Msg("The recipient file is missing for the multipart file.")
			continue
		}

		filteredSolutionArchives = append(filteredSolutionArchives, solutionArchive)
	}

	return filteredSolutionArchives
}

// --- shared helpers (duplicated with multipartuploader) ---

func (f *FileSystem) genBucketPath(bucketName string) string {
	return filepath.Join(f.solutionArchivesLocation, library.FileSystemBucketPrefix+bucketName)
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
