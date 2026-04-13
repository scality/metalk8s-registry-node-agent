package service

type ArchiveCleaner interface {
	// CleanUnusedSolutionArchives cleans unused solution archives from the storage.
	CleanUnusedSolutionArchives(path string, isDir bool) error

	// CleanUnusedSolutions cleans unused solutions from the storage.
	CleanUnusedSolutions(path string, isDir bool) error
}
