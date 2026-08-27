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
	mountWatcher             service.MountWatcher
}

func NewFileSystem(
	logger *slog.Logger,
	solutionArchivesLocation string,
	solutionsLocation string,
	archiveRemover service.ArchiveRemover,
	watcher service.FileWatcher,
	mountWatcher service.MountWatcher,
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
		mountWatcher:             mountWatcher,
	}
}

var _ service.ArchiveMounter = &FileSystem{}

// MountFile mounts the solution archive, identifed by its filename,
// into the solution location.
func (f *FileSystem) MountFile(fileName string, mountPoint string) error {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(92),
			errors.WithDetail("unexpected error while enforcing naming conventions before mounting"),
			errors.WithProperty("file_name", fileName),
		)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(93),
			errors.WithDetail("unexpected error while checking solution archive before mounting"),
			errors.WithProperty("file_name", fileName),
		)
	}

	mountPath := filepath.Join(f.solutionsLocation, mountPoint)
	if err := library.CheckDir(mountPath); err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return errors.Wrap(err,
				errors.WithIdentifier(94),
				errors.WithDetail("unexpected error while checking mount point availability before mounting"),
				errors.WithProperty("mount_path", mountPath),
			)
		}
		if err := os.MkdirAll(mountPath, library.FileSystemDefaultDirMode); err != nil {
			return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
				errors.WithIdentifier(95),
				errors.WithDetail("failed to create mount point directory"),
				errors.WithProperty("mount_path", mountPath),
				errors.CausedBy(err),
			)
		}
	}

	// Add a watcher for the solution path
	solutionName := strings.Split(mountPoint, "/")[0]
	solutionPath := filepath.Join(f.solutionsLocation, solutionName)
	err := f.watcher.AddWatchFileOrDirectory(solutionPath)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(96),
			errors.WithDetail("failed to add solution path to watcher"),
			errors.WithProperty("solution_path", solutionPath),
		)
	}

	// Add a watcher for the mount path
	err = f.watcher.AddWatchFileOrDirectory(mountPath)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(97),
			errors.WithDetail("failed to add mount path to watcher"),
			errors.WithProperty("mount_path", mountPath),
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
				return errors.Wrap(err,
					errors.WithIdentifier(98),
					errors.WithDetail("failed to unmount file before mounting again"),
					errors.WithProperty("mount_path", mountPath),
				)
			}
		} else if errors.Is(err, domain.ErrMountSolutionArchiveNotEmptyDir) {
			// Clean directory and create it again
			if err := os.RemoveAll(mountPath); err != nil {
				return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
					errors.WithIdentifier(99),
					errors.WithDetail("failed to remove mount point directory before mounting again"),
					errors.WithProperty("mount_path", mountPath),
					errors.CausedBy(err),
				)
			}
			if err := os.MkdirAll(mountPath, library.FileSystemDefaultDirMode); err != nil {
				return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
					errors.WithIdentifier(100),
					errors.WithDetail("failed to create mount point directory before mounting again"),
					errors.WithProperty("mount_path", mountPath),
					errors.CausedBy(err),
				)
			}
		} else {
			return errors.Wrap(err,
				errors.WithIdentifier(101),
				errors.WithDetail("failed to determine if the file is mounted"),
				errors.WithProperty("file_path", filePath),
				errors.WithProperty("mount_path", mountPath),
			)
		}
	}

	if !isMounted {
		err := library.MountISO(filePath, mountPath)
		if err != nil {
			if errors.Is(err, unix.EINVAL) {
				// The archive file is not a valid ISO file and is therefore deleted.
				err := f.archiveRemover.DeleteFile(fileName)
				if err != nil {
					return errors.Wrap(err,
						errors.WithIdentifier(102),
						errors.WithDetail("failed to delete the non-valid archive file"),
					)
				}
				return errors.Wrap(domain.ErrMountSolutionArchiveInvalidISO,
					errors.WithIdentifier(103),
					errors.WithDetail("the archive file is not a valid ISO file and is therefore deleted"),
					errors.WithProperty("file_path", filePath),
				)
			}

			return errors.Wrap(err,
				errors.WithIdentifier(104),
				errors.WithDetail("unexpected error while mounting the solution archive"),
				errors.WithProperty("file_path", filePath),
				errors.WithProperty("mount_path", mountPath),
			)
		}
	}

	f.mountWatcher.RecordMount(mountPath, mountPoint)

	return nil
}

// UnmountFile unmounts a file from the storage.
func (f *FileSystem) UnmountFile(mountPoint string) error {
	mountPath := filepath.Join(f.solutionsLocation, mountPoint)
	if err := library.CheckDir(mountPath); err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return errors.Wrap(err,
				errors.WithIdentifier(105),
				errors.WithDetail("unexpected error while checking solution archive before unmounting"),
				errors.WithProperty("mount_path", mountPath),
			)
		}
		return nil
	}

	_, err := library.GetLoopDeviceForMount(mountPath)
	// In case of NotFound error, we want to delete the mount path.
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return errors.Wrap(err,
			errors.WithIdentifier(106),
			errors.WithDetail("unexpected error while getting loop device for unmounting"),
			errors.WithProperty("mount_path", mountPath),
		)
	}
	if err == nil {
		err := library.UnmountISO(mountPath)
		if err != nil {
			return errors.Wrap(err,
				errors.WithIdentifier(107),
				errors.WithDetail("unexpected error while unmounting the solution archive"),
				errors.WithProperty("mount_path", mountPath),
			)
		}
	}

	f.mountWatcher.RecordUnmount(mountPath)

	err = os.RemoveAll(mountPath)
	if err != nil {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(108),
			errors.WithDetail("failed to remove the solution archive mount path"),
			errors.WithProperty("mount_path", mountPath),
			errors.CausedBy(err),
		)
	}

	return nil
}
