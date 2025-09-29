package service

import (
	"os"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

// StorageProvider encapsulates all storage methods required by the Artesca
// Artifacts Upload feature.
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
	Start(filenameCh chan string) error

	// Stop stops the watcher on the storage provider.
	Stop() error

	// ListFiles lists all the flat files in the root location of the storage
	// and returns their names.
	ListFiles() ([]string, error)

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
		artifactMeta *domain.Artifact,
	) (*domain.ArtifactStatus, error)

	// GetMultipartFile retrieves the multipart file recipient from a given
	// bucket and returns its ArtifactMeta.
	GetMultipartFile(bucketName string) (*domain.Artifact, error)

	// GetMultipartFileStatus retrieves the ArtifactStatus of a multipart file
	// recipient based on bucketName and artifactMeta.
	GetMultipartFileStatus(
		bucketName string,
		artifactMeta *domain.Artifact,
	) (*domain.ArtifactStatus, error)

	// DeleteMultipartFile deletes a multipart file recipient from a bucket
	// based in given bucketName and artifactMeta.
	DeleteMultipartFile(bucketName string, artifactMeta *domain.Artifact) error

	// WritePartToMultipartFile properly writes the content of the given part
	// into the multipart file recipient on the bucket indicated by the given
	// bucketName.
	WritePartToMultipartFile(bucketName string, part *domain.Part) (*domain.ArtifactStatus, error)

	// ConsolidateMultipartFile consolidates all the parts of a multipart file
	// in a single flat file into the same bucket it is located.
	//
	// The resulting fileName is the artifactMeta.FileName appended with the
	// artifactMeta.Version. The file extension is properly moved to the end.
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
		artifactMeta *domain.Artifact,
		perm os.FileMode,
	) error

	// GetHashFromFileInfos retrieves the hash of a file from the storage
	// backend.
	GetHashFromFileInfos(fileName string) (string, error)
}
