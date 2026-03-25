package filemounter

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

var _ service.FileMounter = &FileSystem{}

type FileSystem struct {
	logger                   *zerolog.Logger
	solutionArchivesLocation string
	solutionsLocation        string
	fileRemover              service.FileRemover
	fileWatcher              service.FileWatcher
	mountLocks               sync.Map
}

func NewFileSystem(
	logger *zerolog.Logger,
	solutionArchivesLocation string,
	solutionsLocation string,
	fileRemover service.FileRemover,
	fileWatcher service.FileWatcher,
) *FileSystem {
	l := logger.With().
		Str("infrastructure", "file_mounter").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
		solutionsLocation:        solutionsLocation,
		fileRemover:              fileRemover,
		fileWatcher:              fileWatcher,
	}
}

func (f *FileSystem) lockMountPoint(mountPoint string) func() {
	val, _ := f.mountLocks.LoadOrStore(mountPoint, &sync.Mutex{})
	mu := val.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func (f *FileSystem) MountFile(fileName string, mountPoint string) error {
	unlock := f.lockMountPoint(mountPoint)
	defer unlock()

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

	solutionName := strings.Split(mountPoint, "/")[0]
	solutionPath := filepath.Join(f.solutionsLocation, solutionName)
	err := f.fileWatcher.AddWatchFileOrDirectory(solutionPath)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to add solution path to watcher").
			WithProperty("solution_path", solutionPath).
			Throw()
	}

	err = f.fileWatcher.AddWatchFileOrDirectory(mountPath)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to add mount path to watcher").
			WithProperty("mount_path", mountPath).
			Throw()
	}

	isMounted, err := library.IsMounted(filePath, mountPath)
	if err != nil {
		if errors.Is(err, domain.ErrMountSolutionArchiveIncorrectMount) {
			err := f.UnmountFile(mountPoint)
			if err != nil {
				return errors.Stamp(err)
			}
		} else if errors.Is(err, domain.ErrMountSolutionArchiveNotEmptyDir) {
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
				err := f.fileRemover.DeleteFile(fileName)
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

func (f *FileSystem) UnmountFile(mountPoint string) error {
	unlock := f.lockMountPoint(mountPoint)
	defer unlock()

	mountPath := filepath.Join(f.solutionsLocation, mountPoint)
	if err := library.CheckDir(mountPath); err != nil {
		if !errors.Is(err, errors.Intercept(domain.ErrNotFound).WithIdentifier(404000).Throw()) {
			return errors.Stamp(err)
		}
		return nil
	}

	_, err := library.GetLoopDeviceForMount(mountPath)
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
