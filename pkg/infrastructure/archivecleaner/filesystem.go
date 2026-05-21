package archivecleaner

import (
	"context"
	"log/slog"
	"os"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type FileSystem struct {
	logger         *slog.Logger
	store          service.StorageProvider
	archiveMounter service.ArchiveMounter
}

func NewFileSystem(
	logger *slog.Logger,
	store service.StorageProvider,
	archiveMounter service.ArchiveMounter,
) service.ArchiveCleaner {
	return &FileSystem{
		logger: logger.With(
			slog.String("infrastructure", "solution_archives_cleaner"),
			slog.String("implementation", "filesystem"),
		),
		store:          store,
		archiveMounter: archiveMounter,
	}
}

var _ service.ArchiveCleaner = &FileSystem{}

func (f *FileSystem) CleanUnusedSolutionArchives(ctx context.Context, path string, isDir bool) error {
	if isDir {
		err := os.RemoveAll(path)
		if err != nil {
			return errors.From(domain.ErrSolutionArchiveCleanerInternal).
				CausedBy(err).
				WithDetail("failed to delete unused solution archive directory").
				WithProperty("path", path).
				Throw()
		}

		f.logger.DebugContext(ctx, "finished to clean unused solution archive directory",
			slog.String("path", path),
		)

		return nil
	}

	err := library.DeleteFile(path)
	if err != nil {
		return errors.From(domain.ErrSolutionArchiveCleanerInternal).
			CausedBy(err).
			WithDetail("failed to delete unused solution archive").
			WithProperty("path", path).
			Throw()
	}

	f.logger.DebugContext(ctx, "finished to clean unused solution archive",
		slog.String("path", path),
	)

	return nil
}

func (f *FileSystem) CleanUnusedSolutions(ctx context.Context, path string, isDir bool) error {
	if isDir {
		// Unmount the solution
		err := f.archiveMounter.UnmountFile(path)
		if err != nil {
			f.logger.ErrorContext(ctx, "failed to unmount unused solution",
				slog.String("mount_point", path),
				slog.Any("error", err),
			)
			// Continue to try to delete the directory anyway
		}

		// Delete the directory
		err = os.RemoveAll(path)
		if err != nil {
			return errors.From(domain.ErrSolutionArchiveCleanerInternal).
				CausedBy(err).
				WithDetail("failed to delete unused solution directory").
				WithProperty("path", path).
				Throw()
		}

		f.logger.DebugContext(ctx, "finished to clean unused solution directory",
			slog.String("path", path),
		)

		return nil
	}

	err := library.DeleteFile(path)
	if err != nil {
		return errors.From(domain.ErrSolutionArchiveCleanerInternal).
			CausedBy(err).
			WithDetail("failed to delete unused solution").
			WithProperty("path", path).
			Throw()
	}

	f.logger.DebugContext(ctx, "finished to clean unused solution",
		slog.String("path", path),
	)

	return nil
}
