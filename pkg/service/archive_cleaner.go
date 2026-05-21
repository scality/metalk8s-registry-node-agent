package service

import "context"

type ArchiveCleaner interface {
	// CleanUnusedSolutionArchives cleans unused solution archives from the storage.
	CleanUnusedSolutionArchives(ctx context.Context, path string, isDir bool) error

	// CleanUnusedSolutions cleans unused solutions from the storage.
	CleanUnusedSolutions(ctx context.Context, path string, isDir bool) error
}
