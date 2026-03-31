package service

// ArchiveRemover deletes solution archives.
type ArchiveRemover interface {
	// DeleteFile deletes a file from the root location in the storage based on
	// its fileName.
	DeleteFile(fileName string) error
}
