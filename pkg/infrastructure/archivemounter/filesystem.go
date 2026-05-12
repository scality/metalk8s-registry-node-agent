package archivemounter

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
	"golang.org/x/sys/unix"
)

type FileSystem struct {
	logger                   *zerolog.Logger
	solutionArchivesLocation string
	solutionsLocation        string
	archiveRemover           service.ArchiveRemover
	watcher                  service.FileWatcher
}

func NewFileSystem(
	logger *zerolog.Logger,
	solutionArchivesLocation string,
	solutionsLocation string,
	archiveRemover service.ArchiveRemover,
	watcher service.FileWatcher,
) service.ArchiveMounter {
	l := logger.With().
		Str("infrastructure", "archive_mounter").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
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
		return errors.Wrap(err)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return errors.Wrap(err)
	}

	mountPath := filepath.Join(f.solutionsLocation, mountPoint)
	if err := library.CheckDir(mountPath); err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return errors.Wrap(err)
		}
		if err := os.MkdirAll(mountPath, library.FileSystemDefaultDirMode); err != nil {
			return errors.Wrap(err)
		}
	}

	// Add a watcher for the solution path
	solutionName := strings.Split(mountPoint, "/")[0]
	solutionPath := filepath.Join(f.solutionsLocation, solutionName)
	err := f.watcher.AddWatchFileOrDirectory(solutionPath)
	if err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("failed to add solution path to watcher"),
			errors.WithProperty("solution_path", solutionPath),
			errors.CausedBy(err),
		)
	}

	// Add a watcher for the mount path
	err = f.watcher.AddWatchFileOrDirectory(mountPath)
	if err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("failed to add mount path to watcher"),
			errors.WithProperty("mount_path", mountPath),
			errors.CausedBy(err),
		)
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
				return errors.Wrap(err)
			}
		} else if errors.Is(err, domain.ErrMountSolutionArchiveNotEmptyDir) {
			// Clean directory and create it again
			if err := os.RemoveAll(mountPath); err != nil {
				return errors.Wrap(err)
			}
			if err := os.MkdirAll(mountPath, library.FileSystemDefaultDirMode); err != nil {
				return errors.Wrap(err)
			}
		} else {
			return errors.Wrap(err)
		}
	}

	if !isMounted {
		err := library.MountISO(filePath, mountPath)
		if err != nil {
			if errors.Is(err, unix.EINVAL) {
				// The archive file is not a valid ISO file and is therefore deleted.
				err := f.archiveRemover.DeleteFile(fileName)
				if err != nil {
					return errors.Wrap(err)
				}
				return errors.Wrap(domain.ErrMountSolutionArchiveInvalidISO,
					errors.WithIdentifier(400000),
					errors.WithProperty("file_path", filePath),
				)
			}

			return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
				errors.WithIdentifier(500000),
				errors.WithDetail("unexpected error while mounting the file"),
				errors.WithProperty("file_path", filePath),
				errors.WithProperty("mount_path", mountPath),
				errors.CausedBy(err),
			)
		}
	}

	return nil
}

// UnmountFile unmounts a file from the storage.
func (f *FileSystem) UnmountFile(mountPoint string) error {
	mountPath := filepath.Join(f.solutionsLocation, mountPoint)
	if err := library.CheckDir(mountPath); err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return errors.Wrap(err)
		}
		return nil
	}

	_, err := library.GetLoopDeviceForMount(mountPath)
	// In case of NotFound error, we want to delete the mount path.
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return errors.Wrap(err)
	}
	if err == nil {
		err := library.UnmountISO(mountPath)
		if err != nil {
			return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
				errors.WithIdentifier(500000),
				errors.WithDetail("unexpected error while unmounting the file"),
				errors.WithProperty("mount_path", mountPath),
				errors.CausedBy(err),
			)
		}
	}

	err = os.RemoveAll(mountPath)
	if err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(500000),
			errors.WithDetail("failed to remove the mount path"),
			errors.WithProperty("mount_path", mountPath),
			errors.CausedBy(err),
		)
	}

	return nil
}
