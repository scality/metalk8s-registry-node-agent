package filewatcher

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type FileSystem struct {
	sync.WaitGroup
	logger                   *slog.Logger
	watcher                  *fsnotify.Watcher
	store                    service.StorageProvider
	solutionArchivesLocation string
	solutionsLocation        string
}

func NewFileSystem(
	logger *slog.Logger,
	store service.StorageProvider,
	solutionArchivesLocation string,
	solutionsLocation string,
) service.FileWatcher {
	return &FileSystem{
		logger: logger.With(
			slog.String("infrastructure", "file_watcher"),
			slog.String("implementation", "filesystem"),
		),
		store:                    store,
		solutionArchivesLocation: solutionArchivesLocation,
		solutionsLocation:        solutionsLocation,
	}
}

var _ service.FileWatcher = &FileSystem{}

func (f *FileSystem) Init() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.From(domain.ErrStorageProviderInit).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to create watcher").
			Throw()
	}
	f.watcher = watcher
	return nil
}

// AddWatchFileOrDirectory adds a file or directory to the watcher.
func (f *FileSystem) AddWatchFileOrDirectory(path string) error {
	return f.watcher.Add(path)
}

// RemoveWatchFileOrDirectory removes a file or directory from the watcher.
func (f *FileSystem) RemoveWatchFileOrDirectory(path string) error {
	return f.watcher.Remove(path)
}

func (f *FileSystem) watchFiles(filenameChan chan domain.FileEventDetails) {
	defer f.Done()

	for {
		select {
		case e, ok := <-f.watcher.Events:
			if !ok {
				f.logger.Warn("watcher events channel closed")
				return
			}
			log := f.logger.With(
				slog.String("source_event_type", e.Op.String()),
				slog.String("file_name", e.Name),
			)

			log.Debug("event received")

			if err := f.store.RefreshWatchedFileInfos(); err != nil {
				log.Error("failed to update watched file infos", slog.Any("error", err))
			}

			// Determine the origin of the object
			var origin domain.FileOrigin
			if strings.HasPrefix(e.Name, f.solutionArchivesLocation) {
				origin = domain.SolutionArchivesOrigin
			} else if strings.HasPrefix(e.Name, f.solutionsLocation) {
				origin = domain.SolutionsOrigin
			}

			isDir, err := isDirectory(origin, e)
			if err != nil {
				if errors.Is(err, domain.ErrStorageProviderNotFound) {
					log.Debug("object not found to determine if it is a directory", slog.Any("error", err))
					continue
				}
				log.Warn("failed to determine if the object is a directory", slog.Any("error", err))
				// In our use case, we expect to do nothing if the object is undetermined
				continue
			}

			var objectNameVersion string
			switch origin {
			case domain.SolutionArchivesOrigin:
				if isDir {
					// We consider that the directory is a working bucket
					objectName, _ := filepath.Rel(f.solutionArchivesLocation, e.Name)
					objectNameVersion, _ = strings.CutPrefix(objectName, library.FileSystemBucketPrefix)
				} else {
					// We consider that the file is a solution archive file
					fileName := filepath.Base(e.Name)
					objectNameVersion = strings.TrimSuffix(fileName, ".iso")
				}
			case domain.SolutionsOrigin:
				/* objectName should be:
				 * <solution>: for a solution directory
				 * <solution>/<version>: for a mount point directory
				 */
				objectNameVersion, _ = filepath.Rel(f.solutionsLocation, e.Name)
			}

			log.Debug("creating FileEventDetails")
			filenameChan <- domain.FileEventDetails{
				FullPathName: e.Name,
				ObjectName:   objectNameVersion,
				IsDir:        isDir,
				Origin:       origin,
				EventType:    e.Op.String(),
			}
			log.Debug("FileEventDetails created")
		case err, ok := <-f.watcher.Errors:
			if !ok {
				f.logger.Warn("watcher errors channel closed")

				return
			}

			f.logger.Error("watcher error", slog.Any("error", err))
		}
	}
}

func (f *FileSystem) StartWatchFiles(filenameChan chan domain.FileEventDetails) error {
	if err := f.store.InitWatchedFileInfos(); err != nil {
		return errors.Stamp(err)
	}

	f.Add(1)

	go f.watchFiles(filenameChan)

	if err := f.watcher.Add(f.solutionArchivesLocation); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to add solution archive location to watcher").
			WithProperty("solution_archive_location", f.solutionArchivesLocation).
			Throw()
	}

	if err := f.watcher.Add(f.solutionsLocation); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to add solutions location to watcher").
			WithProperty("solutions_location", f.solutionsLocation).
			Throw()
	}

	// Below, we clean the fake objects created before the controller started
	dirEntries, err := os.ReadDir(f.solutionArchivesLocation)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to read solution archives location").
			WithProperty("solution_archives_location", f.solutionArchivesLocation).
			Throw()
	}
	go func(dEntries []os.DirEntry) {
		for _, dirEntry := range dEntries {
			f.logger.Debug("creating an event for object " + dirEntry.Name())
			if !strings.HasPrefix(dirEntry.Name(), f.store.ControlDir()) {
				f.watcher.Events <- fsnotify.Event{
					Name: filepath.Join(f.solutionArchivesLocation, dirEntry.Name()),
					Op:   fsnotify.Create,
				}
			}
		}
	}(dirEntries)

	dirEntriesToAnalyze := []string{}
	dirEntries, err = os.ReadDir(f.solutionsLocation)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to read solutions location").
			WithProperty("solutions_location", f.solutionsLocation).
			Throw()
	}
	for _, dirEntry := range dirEntries {
		dirEntriesToAnalyze = append(dirEntriesToAnalyze, filepath.Join(f.solutionsLocation, dirEntry.Name()))
		if dirEntry.IsDir() {
			subDirEntries, err := os.ReadDir(filepath.Join(f.solutionsLocation, dirEntry.Name()))
			if err != nil {
				return errors.From(domain.ErrStorageProviderInternal).
					WithIdentifier(500000).
					CausedBy(err).
					WithDetail("failed to read sub directory").
					WithProperty("sub_directory", filepath.Join(f.solutionsLocation, dirEntry.Name())).
					Throw()
			}
			for _, subDirEntry := range subDirEntries {
				dirEntriesToAnalyze = append(
					dirEntriesToAnalyze,
					filepath.Join(f.solutionsLocation, dirEntry.Name(), subDirEntry.Name()),
				)
			}
		}

		continue
	}
	go func(dEntries []string) {
		for _, dirEntry := range dEntries {
			f.logger.Debug("creating an event for object " + dirEntry)
			f.watcher.Events <- fsnotify.Event{
				Name: dirEntry,
				Op:   fsnotify.Create,
			}
		}
	}(dirEntriesToAnalyze)
	return nil
}

func (f *FileSystem) StopWatchFiles() error {
	defer f.Wait()

	if err := f.watcher.Close(); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to close watcher").
			Throw()
	}

	return nil
}

// isDirectory determines if the object is a directory or a file.
func isDirectory(origin domain.FileOrigin, e fsnotify.Event) (bool, error) {
	/*
		Determine if it is a directory or a file
		Not so easy in case of delete event the object (file/directory) no more exists
		SolutionArchivesOrigin:
		  * Create or Write or Chmod => os.Stat(e.Name), if FileNotFound => continue, ignore the event
		  * Remove or Rename:
		    * ".bucket.<solution>-<version>" => Directory
			* "<solution>-<version>.iso" => File
		SolutionOrigin:
		  * Create or Write or Chmod => os.Stat(e.Name), if FileNotFound => continue, ignore the event
		  * Remove or Rename: Directory
	*/
	switch e.Op {
	case fsnotify.Remove, fsnotify.Rename:
		if origin == domain.SolutionArchivesOrigin {
			fileName := filepath.Base(e.Name)
			if strings.HasPrefix(fileName, library.FileSystemBucketPrefix) {
				return true, nil
			} else if strings.HasSuffix(fileName, ".iso") {
				return false, nil
			}
			// We cannot determine if it is a directory or a file
			// We raise an error to the caller to handle it
			return false, errors.From(domain.ErrStorageProviderInternal).
				WithDetail("failed to determine if the object is a directory").
				WithProperty("object_name", e.Name).
				WithProperty("origin", origin).
				WithProperty("event_type", e.Op.String()).
				Throw()
		}
		// We consider that the directory is a solution directory
		// In that case, we consider that it is a directory
		return true, nil

	default:
		fileEventStat, err := os.Stat(e.Name)
		if err != nil {
			if os.IsNotExist(err) {
				return false, errors.From(domain.ErrStorageProviderNotFound).
					WithDetail("object not found to determine if it is a directory").
					WithProperty("object_name", e.Name).
					WithProperty("origin", origin).
					WithProperty("event_type", e.Op.String()).
					Throw()
			}
			return false, errors.From(domain.ErrStorageProviderInternal).
				WithDetail("failed to determine if the object is a directory").
				WithProperty("object_name", e.Name).
				WithProperty("origin", origin).
				WithProperty("event_type", e.Op.String()).
				CausedBy(err).
				Throw()
		}
		return fileEventStat.IsDir(), nil
	}
}
