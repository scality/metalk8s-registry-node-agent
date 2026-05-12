package multipartinspector

import (
	"encoding/binary"
	"encoding/json"
	"io"
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
}

func NewFileSystem(
	logger *zerolog.Logger,
	solutionArchivesLocation string,
) service.MultipartInspector {
	l := logger.With().
		Str("infrastructure", "multipart_inspector").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
	}
}

var _ service.MultipartInspector = &FileSystem{}

// GetMultipartFile retrieves the multipart file recipient from a given bucket
// and returns its Solution Archive.
func (f *FileSystem) GetMultipartFile(bucketName string) (*domain.SolutionArchive, error) {
	bucketPath := library.GenBucketPath(f.solutionArchivesLocation, bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return nil, errors.Wrap(err)
	}

	solutionArchiveMeta, err := f.getSolutionArchiveMeta(bucketPath)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return solutionArchiveMeta, nil
}

// GetMultipartFileStatus retrieves the Solution Archive Status of a multipart file recipient
// based on bucketName and solutionArchiveMeta.
func (f *FileSystem) GetMultipartFileStatus(
	bucketName string,
	solutionArchive *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	if err := library.CheckDir(
		library.GenBucketPath(f.solutionArchivesLocation, bucketName),
	); err != nil {
		return nil, errors.Wrap(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := library.GenMultipartFilePaths(
		f.solutionArchivesLocation,
		bucketName,
		solutionArchive.Name,
	)

	if err := library.CheckFile(metaFilePath); err != nil {
		return nil, errors.Wrap(err)
	}

	if err := library.CheckFile(partsFilePath); err != nil {
		return nil, errors.Wrap(err)
	}

	if err := library.CheckFile(recipientFilePath); err != nil {
		return nil, errors.Wrap(err)
	}

	metaFile, err := library.GetFile(metaFilePath)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	defer metaFile.Close() // nolint: errcheck // No error check on defer.

	partsFile, err := library.GetFile(partsFilePath)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	defer partsFile.Close() // nolint: errcheck // No error check on defer.

	var storedSolutionArchiveMeta domain.SolutionArchive
	if err := json.NewDecoder(metaFile).Decode(&storedSolutionArchiveMeta); err != nil {
		return nil, errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to load the metadata of the multipart file"),
			errors.WithProperty("file_path", metaFilePath),
			errors.WithProperty("while", "decoding the metadata from json"),
			errors.CausedBy(err),
		)
	}

	if err := library.CompareSolutionArchiveMetas(solutionArchive, &storedSolutionArchiveMeta); err != nil {
		return nil, errors.Wrap(err)
	}

	partMetas := make(map[int64]*domain.PartMeta)
	for {
		var partMeta domain.PartMeta

		err = binary.Read(partsFile, binary.LittleEndian, &partMeta)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, errors.Wrap(domain.ErrStorageProviderInternal,
				errors.WithIdentifier(500000),
				errors.WithDetail("unable to load the parts metadata of the multipart file"),
				errors.WithProperty("file_path", partsFilePath),
				errors.WithProperty("while", "decoding the parts metadata from binary"),
				errors.CausedBy(err),
			)
		}

		partMetas[partMeta.Start] = &partMeta
	}

	return &domain.SolutionArchiveStatus{
		SolutionArchive: solutionArchive,
		Parts:           partMetas,
	}, nil
}

// getSolutionArchiveMeta returns the Solution Archive Meta instance
// in the bucket with the given bucketPath.
func (f *FileSystem) getSolutionArchiveMeta(bucketPath string) (*domain.SolutionArchive, error) {
	metaFileNames, err := library.ListDirContentNames(bucketPath, library.MultipartMetaFilter)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	solutionArchiveMetas := make([]*domain.SolutionArchive, 0, len(metaFileNames))

	for _, metaFileName := range metaFileNames {
		var solutionArchiveMeta domain.SolutionArchive

		if err := loadSolutionArchiveMeta(&solutionArchiveMeta, filepath.Join(bucketPath, metaFileName)); err != nil {
			return nil, errors.Wrap(err)
		}

		solutionArchiveMetas = append(solutionArchiveMetas, &solutionArchiveMeta)
	}

	solutionArchiveMetas = f.filterOrphansMeta(bucketPath, solutionArchiveMetas)
	if len(solutionArchiveMetas) < 1 {
		return nil, errors.Wrap(domain.ErrStorageProviderNotFound,
			errors.WithIdentifier(404000),
			errors.WithDetail("no multipart files found in the bucket"),
			errors.WithProperty("bucket_path", bucketPath),
		)
	}
	if len(solutionArchiveMetas) > 1 {
		return nil, errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("too many multipart files found in the bucket"),
			errors.WithProperty("bucket_path", bucketPath),
		)
	}

	return solutionArchiveMetas[0], nil
}

// filterOrphansMeta filters the orphaned SolutionArchiveMeta instances from the given
// solutionArchiveMetas list and returns the filtered list.
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

// loadSolutionArchiveMeta loads the Solution Archive Meta instance from
// the multipart meta file with the given file path.
func loadSolutionArchiveMeta(meta *domain.SolutionArchive, filePath string) error {
	metaFile, err := library.GetFile(filePath)
	if err != nil {
		return errors.Wrap(err)
	}

	defer metaFile.Close() // nolint: errcheck // No error check on defer.

	if err := json.NewDecoder(metaFile).Decode(meta); err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to load the metadata of the multipart file"),
			errors.WithProperty("file_path", filePath),
			errors.WithProperty("while", "decoding the metadata from json"),
			errors.CausedBy(err),
		)
	}

	return nil
}
