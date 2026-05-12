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
	if err := library.CheckDir(library.GenBucketPath(f.solutionArchivesLocation, bucketName)); err != nil {
		return errors.Wrap(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := library.GenMultipartFilePaths(
		f.solutionArchivesLocation,
		bucketName,
		solutionArchive.Name,
	)

	if err := library.CheckFile(metaFilePath); err != nil {
		return errors.Wrap(err)
	}

	if err := library.CheckFile(partsFilePath); err != nil {
		return errors.Wrap(err)
	}

	if err := library.CheckFile(recipientFilePath); err != nil {
		return errors.Wrap(err)
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
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to delete the multipart file"),
			errors.WithProperty("problems", problems),
		)
	}

	return nil
}
