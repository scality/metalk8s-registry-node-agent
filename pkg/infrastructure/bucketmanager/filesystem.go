package bucketmanager

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
) service.BucketManager {
	l := logger.With().
		Str("infrastructure", "bucket_manager").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
	}
}

var _ service.BucketManager = &FileSystem{}

// CreateBucket creates a new bucket in the storage.
func (f *FileSystem) CreateBucket(bucketName string) error {
	if err := library.EnforceNamingConventions(bucketName); err != nil {
		return errors.Wrap(err)
	}

	bucketPath := library.GenBucketPath(f.solutionArchivesLocation, bucketName)
	if err := library.CheckDir(bucketPath); err != nil &&
		!errors.Is(err, domain.ErrStorageProviderNotFound) {
		return errors.Wrap(err)
	}

	if err := os.Mkdir(bucketPath, library.FileSystemDefaultDirMode); err != nil && !os.IsExist(err) {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to create the bucket"),
			errors.WithProperty("bucket_path", bucketPath),
			errors.CausedBy(err),
		)
	}

	return nil
}

// ListBuckets lists all the buckets in the storage and returns their bucketNames,
// after having removed FileSystemBucketPrefix.
func (f *FileSystem) ListBuckets() ([]string, error) {
	buckets, err := library.ListDirContentNames(f.solutionArchivesLocation, library.BucketFilter)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if len(buckets) == 0 {
		return nil, errors.Wrap(domain.ErrStorageProviderNotFound,
			errors.WithIdentifier(404000),
			errors.WithDetail("no buckets found"),
		)
	}

	for i, bucket := range buckets {
		buckets[i] = bucket[len(library.FileSystemBucketPrefix):]
	}

	return buckets, nil
}

// DeleteBucket deletes a bucket from the storage.
func (f *FileSystem) DeleteBucket(bucketName string) error {
	if err := library.EnforceNamingConventions(bucketName); err != nil {
		return errors.Wrap(err)
	}

	bucketPath := library.GenBucketPath(f.solutionArchivesLocation, bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return errors.Wrap(err)
	}

	if err := os.RemoveAll(bucketPath); err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("unable to delete the bucket"),
			errors.WithProperty("bucket_path", bucketPath),
			errors.CausedBy(err),
		)
	}

	return nil
}
