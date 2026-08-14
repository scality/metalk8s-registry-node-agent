package multipartinspector

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type FileSystem struct {
	logger                   *slog.Logger
	solutionArchivesLocation string
}

func NewFileSystem(
	logger *slog.Logger,
	solutionArchivesLocation string,
) service.MultipartInspector {
	return &FileSystem{
		logger: logger.With(
			slog.String("infrastructure", "multipart_inspector"),
			slog.String("implementation", "filesystem"),
		),
		solutionArchivesLocation: solutionArchivesLocation,
	}
}

var _ service.MultipartInspector = &FileSystem{}

// GetMultipartFile retrieves the multipart file recipient from a given bucket
// and returns its Solution Archive.
func (f *FileSystem) GetMultipartFile(ctx context.Context, bucketName string) (*domain.SolutionArchive, error) {
	bucketPath := library.GenBucketPath(f.solutionArchivesLocation, bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(140),
			errors.WithDetail("unexpected error while checking bucket directory"),
			errors.WithProperty("bucket_path", bucketPath),
		)
	}

	solutionArchiveMeta, err := f.getSolutionArchiveMeta(ctx, bucketPath)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(141),
			errors.WithDetail("unexpected error while getting solution archive metadata"),
			errors.WithProperty("bucket_path", bucketPath),
		)
	}

	return solutionArchiveMeta, nil
}

// GetMultipartFileStatus retrieves the Solution Archive Status of a multipart file recipient
// based on bucketName and solutionArchiveMeta.
func (f *FileSystem) GetMultipartFileStatus(
	_ context.Context,
	bucketName string,
	solutionArchive *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	bucketPath := library.GenBucketPath(f.solutionArchivesLocation, bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(142),
			errors.WithDetail("unexpected error while checking bucket directory"),
			errors.WithProperty("bucket_path", bucketPath),
		)
	}

	metaFilePath, partsFilePath, recipientFilePath := library.GenMultipartFilePaths(
		f.solutionArchivesLocation,
		bucketName,
		solutionArchive.Name,
	)

	if err := library.CheckFile(metaFilePath); err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(143),
			errors.WithDetail("unexpected error while checking meta file"),
			errors.WithProperty("meta_file_path", metaFilePath),
		)
	}

	if err := library.CheckFile(partsFilePath); err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(144),
			errors.WithDetail("unexpected error while checking parts file"),
			errors.WithProperty("parts_file_path", partsFilePath),
		)
	}

	if err := library.CheckFile(recipientFilePath); err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(145),
			errors.WithDetail("unexpected error while checking recipient file"),
			errors.WithProperty("recipient_file_path", recipientFilePath),
		)
	}

	metaFile, err := library.GetFile(metaFilePath)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(146),
			errors.WithDetail("unexpected error while getting meta file content"),
			errors.WithProperty("meta_file_path", metaFilePath),
		)
	}

	defer metaFile.Close() // nolint: errcheck // No error check on defer.

	partsFile, err := library.GetFile(partsFilePath)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(147),
			errors.WithDetail("unexpected error while getting parts file content"),
			errors.WithProperty("parts_file_path", partsFilePath),
		)
	}

	defer partsFile.Close() // nolint: errcheck // No error check on defer.

	var storedSolutionArchiveMeta domain.SolutionArchive
	if err := json.NewDecoder(metaFile).Decode(&storedSolutionArchiveMeta); err != nil {
		return nil, errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(148),
			errors.WithDetail("unable to decode metadata from JSON in multipart meta file"),
			errors.WithProperty("file_path", metaFilePath),
			errors.CausedBy(err),
		)
	}

	if err := library.CompareSolutionArchiveMetas(solutionArchive, &storedSolutionArchiveMeta); err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(149),
			errors.WithDetail("unexpected error while comparing solution archive metas"),
			errors.WithProperty("solution_archive_name", solutionArchive.Name),
		)
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
				errors.WithIdentifier(150),
				errors.WithDetail("unable to load the parts metadata from binary multipart file"),
				errors.WithProperty("file_path", partsFilePath),
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
func (f *FileSystem) getSolutionArchiveMeta(ctx context.Context, bucketPath string) (*domain.SolutionArchive, error) {
	metaFileNames, err := library.ListDirContentNames(bucketPath, library.MultipartMetaFilter)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(151),
			errors.WithDetail("unexpected error while listing multipart meta files"),
			errors.WithProperty("bucket_path", bucketPath),
		)
	}

	solutionArchiveMetas := make([]*domain.SolutionArchive, 0, len(metaFileNames))

	for _, metaFileName := range metaFileNames {
		var solutionArchiveMeta domain.SolutionArchive

		if err := loadSolutionArchiveMeta(&solutionArchiveMeta, filepath.Join(bucketPath, metaFileName)); err != nil {
			return nil, errors.Wrap(err,
				errors.WithIdentifier(152),
				errors.WithDetail("unexpected error while loading solution archive metadata"),
				errors.WithProperty("meta_file_path", filepath.Join(bucketPath, metaFileName)),
			)
		}

		solutionArchiveMetas = append(solutionArchiveMetas, &solutionArchiveMeta)
	}

	solutionArchiveMetas = f.filterOrphansMeta(ctx, bucketPath, solutionArchiveMetas)
	if len(solutionArchiveMetas) < 1 {
		return nil, errors.Wrap(domain.ErrStorageProviderNotFound,
			errors.WithIdentifier(153),
			errors.WithDetail("no multipart files found in the bucket"),
			errors.WithProperty("bucket_path", bucketPath),
		)
	}
	if len(solutionArchiveMetas) > 1 {
		return nil, errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(154),
			errors.WithDetail("too many multipart files found in the bucket"),
			errors.WithProperty("bucket_path", bucketPath),
		)
	}

	return solutionArchiveMetas[0], nil
}

// filterOrphansMeta filters the orphaned SolutionArchiveMeta instances from the given
// solutionArchiveMetas list and returns the filtered list.
func (f *FileSystem) filterOrphansMeta(
	ctx context.Context,
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
			f.logger.WarnContext(ctx, "The parts file is missing for the multipart file.", slog.Any("error", err))

			continue
		}

		if err := library.CheckFile(
			filepath.Join(
				bucketPath,
				solutionArchive.Name+library.FileSystemMultipartRecipientSuffix,
			),
		); err != nil {
			f.logger.WarnContext(ctx, "The recipient file is missing for the multipart file.", slog.Any("error", err))

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
		return errors.Wrap(err,
			errors.WithIdentifier(155),
			errors.WithDetail("unexpected error while getting multipart meta file content"),
			errors.WithProperty("meta_file_path", filePath),
		)
	}

	defer metaFile.Close() // nolint: errcheck // No error check on defer.

	if err := json.NewDecoder(metaFile).Decode(meta); err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(156),
			errors.WithDetail("unable to decode metadata from JSON in multipart meta file"),
			errors.WithProperty("file_path", filePath),
			errors.CausedBy(err),
		)
	}

	return nil
}
