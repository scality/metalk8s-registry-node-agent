package service

import (
	"io"
	"os"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

// StorageProvider encapsulates all storage methods required by the Solution
// Archives Upload feature.
//
// Implementations MUST be stateless. All methods MUST perform their operations
// directly on the storage background with no information retention. If some
// information is needed to be retained, it MUST stored in the storage backend.
//
// Implementations MUST be thread-safe and provide all necessary locking
// mechanisms to ensure that all methods are safe to be called concurrently.
//
// Implementations MUST have a root storage location where flat files should be
// handled by using the SaveFile, ListFiles, GetFile, and DeleteFile methods.
//
// Implementations MUST have the ability to provide buckets, which are
// identified storage containers. Buckets are a flat structure, so no sub-
// -buckets are allowed.
//
// Implementations MUST have the ability to handle flat file storage into their
// buckets by using the SaveFileToBucket, ListFilesInBucket, GetFileFromBucket,
// and DeleteFileFromBucket methods.
//
// Implementations MUST have the ability to handle multipart file storage into
// their buckets. MultipartFiles are recipients to large files that should be
// uploaded in parts. Those must be handled by using the CreateMultipartFile,
// ListMultipartFiles, GetMultipartFileInfo, DeleteMultipartFile,
// WriteAPartToMultipartFile, and ConsolidateMultipartFile methods.
//
// No multipart file should be handled in the providers root location.
//
// The responsibility for close all io objects is delegated to the caller.
type StorageProvider interface {
	BucketManager
	ArchiveCleaner
	ArchiveMounter
	ArchiveLister
	ArchiveRemover
	MultipartUploader

	// RWLocker primitives:
	Lock()
	Unlock()
	RLock()
	RUnlock()

	// Init initializes the storage provider.
	Init() error

	// Start starts the watcher on the storage provider.
	Start(filenameChan chan domain.FileEventDetails) error

	// Stop stops the watcher on the storage provider.
	Stop() error

	// SaveFile saves a file to the root location in the storage. The fileName
	// MUST be unique relative to the root location.
	//
	// When supported by the storage backend, the perm parameter is used to set
	// the file permissions.
	//
	// The caller should close the content reader as soon as possible after
	// this method returns.
	SaveFile(fileName string, content io.Reader, perm os.FileMode) error

	// GetFile retrieves the content of a file from the root location in the
	// storage based on its fileName.
	//
	// The caller should close the content reader as early as possible.
	GetFile(fileName string) (io.ReadCloser, error)

	// HashFile calculates the hash of a file from the root location in the
	// storage based on its fileName.
	HashFile(fileName string) (string, error)

	// GetMultipartFile retrieves the multipart file recipient from a given
	// bucket and returns its SolutionArchiveMeta.
	GetMultipartFile(bucketName string) (*domain.SolutionArchive, error)

	// GetMultipartFileStatus retrieves the SolutionArchiveStatus of a multipart file
	// recipient based on bucketName and solutionArchiveMeta.
	GetMultipartFileStatus(
		bucketName string,
		solutionArchiveMeta *domain.SolutionArchive,
	) (*domain.SolutionArchiveStatus, error)

	// DeleteMultipartFile deletes a multipart file recipient from a bucket
	// based in given bucketName and solutionArchiveMeta.
	DeleteMultipartFile(bucketName string, solutionArchiveMeta *domain.SolutionArchive) error

	// AddWatchFileOrDirectory adds a file or directory to the watcher.
	AddWatchFileOrDirectory(path string) error

	// RemoveWatchFileOrDirectory removes a file or directory from the watcher.
	RemoveWatchFileOrDirectory(path string) error
}
