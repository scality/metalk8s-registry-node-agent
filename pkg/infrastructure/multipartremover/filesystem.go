package multipartremover

import (
	"os"

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
) service.MultipartRemover {
	l := logger.With().
		Str("infrastructure", "multipart_remover").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
	}
}

var _ service.MultipartRemover = &FileSystem{}

// DeleteMultipartFile deletes a multipart file recipient from a bucket based on bucketName and solutionArchive.
func (f *FileSystem) DeleteMultipartFile(
	bucketName string,
	solutionArchive *domain.SolutionArchive,
) error {
	bucketPath := library.GenBucketPath(f.solutionArchivesLocation, bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(157),
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
		return errors.Wrap(err,
			errors.WithIdentifier(158),
			errors.WithDetail("unexpected error while checking meta file"),
			errors.WithProperty("meta_file_path", metaFilePath),
		)
	}

	if err := library.CheckFile(partsFilePath); err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(159),
			errors.WithDetail("unexpected error while checking parts file"),
			errors.WithProperty("parts_file_path", partsFilePath),
		)
	}

	if err := library.CheckFile(recipientFilePath); err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(160),
			errors.WithDetail("unexpected error while checking recipient file"),
			errors.WithProperty("recipient_file_path", recipientFilePath),
		)
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
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(161),
			errors.WithDetail("unable to delete multipart file(s)"),
			errors.WithProperties(problems),
		)
	}

	return nil
}
