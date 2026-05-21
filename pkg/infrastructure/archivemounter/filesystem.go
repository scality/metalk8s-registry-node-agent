package archivemounter

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
	"golang.org/x/sys/unix"
)

type FileSystem struct {
	logger                   *slog.Logger
	solutionArchivesLocation string
	solutionsLocation        string
	archiveRemover           service.ArchiveRemover
	watcher                  service.FileWatcher
}

func NewFileSystem(
	logger *slog.Logger,
	solutionArchivesLocation string,
	solutionsLocation string,
	archiveRemover service.ArchiveRemover,
	watcher service.FileWatcher,
) service.ArchiveMounter {
	return &FileSystem{
		logger: logger.With(
			slog.String("infrastructure", "archive_mounter"),
			slog.String("implementation", "filesystem"),
		),
		solutionArchivesLocation: solutionArchivesLocation,
		solutionsLocation:        solutionsLocation,
		archiveRemover:           archiveRemover,
		watcher:                  watcher,
	}
}

var _ service.ArchiveMounter = &FileSystem{}

// MountFile mounts the solution archive, identifed by its filename,
// into the solution location.
func (f *FileSystem) MountFile(fileName string, mountPoint string) error {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return errors.Stamp(err)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return errors.Stamp(err)
	}

	mountPath := filepath.Join(f.solutionsLocation, mountPoint)
	if err := library.CheckDir(mountPath); err != nil {
		if !errors.Is(err, errors.Intercept(domain.ErrNotFound).WithIdentifier(404000).Throw()) {
			return errors.Stamp(err)
		}
		if err := os.MkdirAll(mountPath, library.FileSystemDefaultDirMode); err != nil {
			return errors.Stamp(err)
		}
	}

	// Add a watcher for the solution path
	solutionName := strings.Split(mountPoint, "/")[0]
	solutionPath := filepath.Join(f.solutionsLocation, solutionName)
	err := f.watcher.AddWatchFileOrDirectory(solutionPath)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to add solution path to watcher").
			WithProperty("solution_path", solutionPath).
			Throw()
	}

	// Add a watcher for the mount path
	err = f.watcher.AddWatchFileOrDirectory(mountPath)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to add mount path to watcher").
			WithProperty("mount_path", mountPath).
			Throw()
	}

	// Check if the mount point is correct
	// * No mount point found => False / No error
	// * Correct mount point => True / No error
	// * Other => False / Error
	isMounted, err := library.IsMounted(filePath, mountPath)
	if err != nil {
		if errors.Is(err, domain.ErrMountSolutionArchiveIncorrectMount) {
			// Unmount before mount again
			err := f.UnmountFile(mountPoint)
			if err != nil {
				return errors.Stamp(err)
			}
		} else if errors.Is(err, domain.ErrMountSolutionArchiveNotEmptyDir) {
			// Clean directory and create it again
			if err := os.RemoveAll(mountPath); err != nil {
				return errors.Stamp(err)
			}
			if err := os.MkdirAll(mountPath, library.FileSystemDefaultDirMode); err != nil {
				return errors.Stamp(err)
			}
		} else {
			return errors.Stamp(err)
		}
	}

	if !isMounted {
		err := library.MountISO(filePath, mountPath)
		if err != nil {
			if errors.Is(err, unix.EINVAL) {
				// The archive file is not a valid ISO file and is therefore deleted.
				err := f.archiveRemover.DeleteFile(fileName)
				if err != nil {
					return errors.Stamp(err)
				}
				return errors.From(domain.ErrMountSolutionArchiveInvalidISO).
					WithIdentifier(400000).
					WithProperty("file_path", filePath).
					Throw()
			}

			return errors.From(domain.ErrMountSolutionArchiveInternal).
				WithIdentifier(500000).
				WithDetail("unexpected error while mounting the file").
				WithProperty("file_path", filePath).
				WithProperty("mount_path", mountPath).
				CausedBy(err).
				Throw()
		}
	}

	return nil
}

// UnmountFile unmounts a file from the storage.
func (f *FileSystem) UnmountFile(mountPoint string) error {
	mountPath := filepath.Join(f.solutionsLocation, mountPoint)
	if err := library.CheckDir(mountPath); err != nil {
		if !errors.Is(err, errors.Intercept(domain.ErrNotFound).WithIdentifier(404000).Throw()) {
			return errors.Stamp(err)
		}
		return nil
	}

	_, err := library.GetLoopDeviceForMount(mountPath)
	// In case of NotFound error, we want to delete the mount path.
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return errors.Stamp(err)
	}
	if err == nil {
		err := library.UnmountISO(mountPath)
		if err != nil {
			return errors.From(domain.ErrMountSolutionArchiveInternal).
				WithIdentifier(500000).
				WithDetail("unexpected error while unmounting the file").
				WithProperty("mount_path", mountPath).
				CausedBy(err).
				Throw()
		}
	}

	err = os.RemoveAll(mountPath)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to remove the mount path").
			WithProperty("mount_path", mountPath).
			Throw()
	}

	return nil
}
