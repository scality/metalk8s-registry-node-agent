package archivecleaner

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type FileSystem struct {
	logger         *zerolog.Logger
	store          service.StorageProvider
	archiveMounter service.ArchiveMounter
}

func NewFileSystem(
	logger *zerolog.Logger,
	store service.StorageProvider,
	archiveMounter service.ArchiveMounter,
) service.ArchiveCleaner {
	l := logger.With().
		Str("infrastructure", "solution_archives_cleaner").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:         &l,
		store:          store,
		archiveMounter: archiveMounter,
	}
}

var _ service.ArchiveCleaner = &FileSystem{}

func (f *FileSystem) CleanUnusedSolutionArchives(path string, isDir bool) error {
	if isDir {
		err := os.RemoveAll(path)
		if err != nil {
			return errors.Wrap(domain.ErrSolutionArchiveCleanerInternal,
				errors.WithDetail("failed to delete unused solution archive directory"),
				errors.WithProperty("path", path),
				errors.CausedBy(err),
			)
		}

		f.logger.Debug().
			Str("path", path).
			Msg("finished to clean unused solution archive directory")

		return nil
	}

	err := library.DeleteFile(path)
	if err != nil {
		return errors.Wrap(domain.ErrSolutionArchiveCleanerInternal,
			errors.WithDetail("failed to delete unused solution archive"),
			errors.WithProperty("path", path),
			errors.CausedBy(err),
		)
	}

	f.logger.Debug().
		Str("path", path).
		Msg("finished to clean unused solution archive")

	return nil
}

func (f *FileSystem) CleanUnusedSolutions(path string, isDir bool) error {
	if isDir {
		// Unmount the solution
		err := f.archiveMounter.UnmountFile(path)
		if err != nil {
			f.logger.Error().
				Err(err).
				Str("mount_point", path).
				Msg("failed to unmount unused solution")
			// Continue to try to delete the directory anyway
		}

		// Delete the directory
		err = os.RemoveAll(path)
		if err != nil {
			return errors.Wrap(domain.ErrSolutionArchiveCleanerInternal,
				errors.WithDetail("failed to delete unused solution directory"),
				errors.WithProperty("path", path),
				errors.CausedBy(err),
			)
		}

		f.logger.Debug().
			Str("path", path).
			Msg("finished to clean unused solution directory")

		return nil
	}

	err := library.DeleteFile(path)
	if err != nil {
		return errors.Wrap(domain.ErrSolutionArchiveCleanerInternal,
			errors.WithDetail("failed to delete unused solution"),
			errors.WithProperty("path", path),
			errors.CausedBy(err),
		)
	}

	f.logger.Debug().
		Str("path", path).
		Msg("finished to clean unused solution")

	return nil
}
