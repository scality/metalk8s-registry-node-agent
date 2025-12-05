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

	// ListFiles lists all the flat files in the root location of the storage
	// and returns their names.
	ListFiles() ([]string, error)

	// GetFile retrieves the content of a file from the root location in the
	// storage based on its fileName.
	//
	// The caller should close the content reader as early as possible.
	GetFile(fileName string) (io.ReadCloser, error)

	// DeleteFile deletes a file from the root location in the storage based on
	// its fileName.
	DeleteFile(fileName string) error

	// HashFile calculates the hash of a file from the root location in the
	// storage based on its fileName.
	HashFile(fileName string) (string, error)

	// CreateBucket creates a new bucket in the storage. The bucketName MUST be
	// unique relative to the storage.
	CreateBucket(bucketName string) error

	// ListBuckets lists all the buckets in the storage and returns their
	// bucketNames.
	ListBuckets() ([]string, error)

	// DeleteBucket deletes a bucket, and all its content, from the storage
	// based on its bucketName.
	DeleteBucket(bucketName string) error

	// MoveFileToRoot moves a file from a bucket to the root location in the
	// storage.
	MoveFileToRoot(bucketName, fileName, newFileName string) error

	// CreateMultipartFiles creates into a bucket:
	// - a metadata file
	// - a multipart file recipient
	// - a parts synthesis file
	CreateMultipartFiles(
		bucketName string,
		solutionArchiveMeta *domain.SolutionArchive,
	) (*domain.SolutionArchiveStatus, error)

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

	// WritePartToMultipartFile properly writes the content of the given part
	// into the multipart file recipient on the bucket indicated by the given
	// bucketName.
	WritePartToMultipartFile(bucketName string, part *domain.Part) (*domain.SolutionArchiveStatus, error)

	// ConsolidateMultipartFile consolidates all the parts of a multipart file
	// in a single flat file into the same bucket it is located.
	//
	// The resulting fileName is the solutionArchiveMeta.FileName appended with the
	// solutionArchiveMeta.Version. The file extension is properly moved to the end.
	// Since the multipart file is consolidated, the parts and all metadata
	// associated are no more available.
	//
	// Also, it will o more appears in the ListMultipartFiles method, but in
	// the ListFilesInBucket instead. That way its content becomes accessible.
	//
	// When supported by the storage backend, the perm parameter is used to set
	// the file permissions.
	ConsolidateMultipartFile(
		bucketName string,
		solutionArchiveMeta *domain.SolutionArchive,
		perm os.FileMode,
	) error

	// GetHashFromFileInfos retrieves the hash of a file from the storage
	// backend.
	GetHashFromFileInfos(fileName string) (string, error)

	// GetSizeFromFileInfos retrieves the size of a file from the storage
	// backend.
	GetSizeFromFileInfos(fileName string) (int64, error)

	// MountFile mounts a file into the storage.
	MountFile(fileName string, mountPoint string) error

	// UnmountFile unmounts a file from the storage.
	UnmountFile(mountPoint string) error

	// AddWatchFileOrDirectory adds a file or directory to the watcher.
	AddWatchFileOrDirectory(path string) error

	// RemoveWatchFileOrDirectory removes a file or directory from the watcher.
	RemoveWatchFileOrDirectory(path string) error
}
