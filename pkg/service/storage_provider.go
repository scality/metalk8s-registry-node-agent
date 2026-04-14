package service

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
	// Init initializes the storage provider.
	Init() error

	// GetArchiveSize retrieves the size of a file from the storage
	// backend.
	GetArchiveSize(fileName string) (int64, error)

	// GetArchiveHash retrieves the hash of a file from the storage
	// backend.
	GetArchiveHash(fileName string) (string, error)

	// ControlDir returns the control directory.
	ControlDir() string

	// InitWatchedFileInfos initializes the watched file infos.
	InitWatchedFileInfos() error

	// RefreshWatchedFileInfos refreshes the watched file infos.
	RefreshWatchedFileInfos() error
}
