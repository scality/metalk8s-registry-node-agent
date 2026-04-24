package service

// ArchiveLister enumerates root files and queries cached file metadata.
type ArchiveLister interface {
	// ListFiles lists all the flat files in the root location of the storage
	// and returns their names.
	ListFiles() ([]string, error)

	// GetArchiveSize retrieves the size of a file from the storage
	// backend.
	GetArchiveSize(fileName string) (int64, error)

	// GetArchiveHash retrieves the hash of a file from the storage
	// backend.
	GetArchiveHash(fileName string) (string, error)

	// GetArchiveCRC32Checksum retrieves the CRC32 checksum of a file from the storage
	// backend.
	GetArchiveCRC32Checksum(fileName string) (uint32, error)
}
