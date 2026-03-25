package storageprovider

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
)

type (
	FileSystem struct {
		sync.RWMutex
		sync.WaitGroup

		logger *zerolog.Logger

		solutionArchivesLocation string
		solutionsLocation        string
		interestContentFilter    library.ContentFilter
		watchedFileStore         *WatchedFileStore
		watcher                  *fsnotify.Watcher
	}

	FileOpts struct {
		Logger                     *zerolog.Logger
		SolutionArchivesLocation   string
		SolutionsLocation          string
		InterestContentFilterRegex *regexp.Regexp
	}
)

const (
	controlDir = ".storageprovider"
)

func NewFileSystem(opts *FileOpts) *FileSystem {
	l := opts.Logger.With().
		Str("infrastructure", "storage_provider").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: opts.SolutionArchivesLocation,
		solutionsLocation:        opts.SolutionsLocation,
		interestContentFilter:    library.NewRegexNormalFileFilter(opts.InterestContentFilterRegex),
		watchedFileStore:         NewWatchedFileStore(),
	}
}

func (f *FileSystem) WatchedFileStore() *WatchedFileStore {
	return f.watchedFileStore
}

func (f *FileSystem) Watcher() *fsnotify.Watcher {
	return f.watcher
}

func (f *FileSystem) InterestContentFilter() library.ContentFilter {
	return f.interestContentFilter
}

func (f *FileSystem) SolutionArchivesLocation() string {
	return f.solutionArchivesLocation
}

func (f *FileSystem) SolutionsLocation() string {
	return f.solutionsLocation
}

// Init initializes the storage provider.
func (f *FileSystem) Init() error {
	f.Lock()
	defer f.Unlock()

	if err := os.MkdirAll(f.solutionArchivesLocation, library.FileSystemDefaultDirMode); err != nil {
		return errors.From(domain.ErrStorageProviderInit).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to create solution archive location").
			WithProperty("solution_archive_location", f.solutionArchivesLocation).
			Throw()
	}

	if err := os.MkdirAll(f.solutionsLocation, library.FileSystemDefaultDirMode); err != nil {
		return errors.From(domain.ErrStorageProviderInit).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to create solution location").
			WithProperty("solution_location", f.solutionsLocation).
			Throw()
	}

	controlDirectoryPath := filepath.Join(f.solutionArchivesLocation, controlDir)

	if err := os.MkdirAll(controlDirectoryPath, library.FileSystemDefaultDirMode); err != nil {
		return errors.From(domain.ErrStorageProviderInit).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to create control directory").
			WithProperty("control_directory", controlDirectoryPath).
			Throw()
	}

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

// Start starts the watcher on the storage provider.
func (f *FileSystem) Start(filenameChan chan domain.FileEventDetails) error {
	f.Lock()
	defer f.Unlock()

	return f.startWatchFiles(filenameChan)
}

// Stop stops the watcher on the storage provider.
func (f *FileSystem) Stop() error {
	return f.stopWatchFiles()
}

func (f *FileSystem) SaveFile(
	fileName string,
	content io.Reader,
	perm os.FileMode,
) error {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return errors.Stamp(err)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil && !errors.Is(err,
		errors.Intercept(domain.ErrNotFound).WithIdentifier(404000).Throw()) {
		return errors.Stamp(err)
	}

	if err := library.SaveFile(filePath, content, perm); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

func (f *FileSystem) HashFile(
	fileName string,
) (string, error) {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return "", errors.Stamp(err)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if f.watchedFileStore.IsUpToDate(fileName, filePath) {
		hash, _ := f.watchedFileStore.GetHash(fileName)
		return hash, nil
	}

	if err := library.CheckFile(filePath); err != nil {
		return "", errors.Stamp(err)
	}

	hash, err := library.HashFile(filePath)
	if err != nil {
		return "", errors.Stamp(err)
	}

	return hash, nil
}

// --- watcher subsystem ---

type watchedFilesMap = map[string]*WatchedFileInfo

const watchedFilesInfoName = "watched_files_info.json"

func (f *FileSystem) genControlDirPath() string {
	return filepath.Join(f.solutionArchivesLocation, controlDir)
}

func (f *FileSystem) genWatchedFilesPath() string {
	return filepath.Join(f.genControlDirPath(), watchedFilesInfoName)
}

func (f *FileSystem) genWatchedFileInfo(fileEntry os.DirEntry) (*WatchedFileInfo, error) {
	filePath := filepath.Join(f.solutionArchivesLocation, fileEntry.Name())

	hash, err := library.HashFile(filePath)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	fileInfo, err := fileEntry.Info()
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to get file info").
			WithProperty("file_path", filePath).
			Throw()
	}

	return &WatchedFileInfo{
		Size:          fileInfo.Size(),
		LastChangedAt: fileInfo.ModTime(),
		Hash:          hash,
	}, nil
}

func (f *FileSystem) genWatchedFileInfos(fileEntries []os.DirEntry) watchedFilesMap {
	type watchedFileEntry struct {
		fileName string
		fileInfo *WatchedFileInfo
	}

	watchedFileChan := make(chan *watchedFileEntry, len(fileEntries))

	var wg sync.WaitGroup

	for _, fileEntry := range fileEntries {
		wg.Add(1)

		go func() {
			defer wg.Done()

			watchedFileInfo, err := f.genWatchedFileInfo(fileEntry)
			if err != nil {
				f.logger.Error().Err(err).Msg("failed to generate watched file info")

				return
			}

			f.logger.Debug().
				Str("file_name", fileEntry.Name()).
				Msg("watched file info generated")
			watchedFileChan <- &watchedFileEntry{
				fileName: fileEntry.Name(),
				fileInfo: watchedFileInfo,
			}
		}()
	}

	wg.Wait()

	close(watchedFileChan)

	watchedFiles := make(watchedFilesMap, len(fileEntries))
	for watchedFileEntry := range watchedFileChan {
		watchedFiles[watchedFileEntry.fileName] = watchedFileEntry.fileInfo
	}

	return watchedFiles
}

func (f *FileSystem) loadWatchedFileInfos() (watchedFilesMap, error) {
	watchedFileMap := make(map[string]*WatchedFileInfo)
	watchedFileInfosFile, err := library.GetFile(f.genWatchedFilesPath())
	if err != nil {
		if errors.Is(err,
			errors.Intercept(domain.ErrStorageProviderNotFound).
				WithIdentifier(404000).
				Throw()) {
			return watchedFileMap, nil
		}
		return nil, errors.Stamp(err)
	}

	defer watchedFileInfosFile.Close() //nolint:errcheck

	jsonDecoder := json.NewDecoder(watchedFileInfosFile)
	if err := jsonDecoder.Decode(&watchedFileMap); err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to decode watched file infos").
			Throw()
	}

	return watchedFileMap, nil
}

func (f *FileSystem) saveWatchedFileInfos(watchedFileInfos watchedFilesMap) error {
	f.watchedFileStore.Set(watchedFileInfos)

	watchedFileInfosBytes, err := json.Marshal(watchedFileInfos)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to marshal watched file infos").
			Throw()
	}

	if err := library.SaveFile(
		f.genWatchedFilesPath(),
		bytes.NewBuffer(watchedFileInfosBytes),
		library.FileSystemDefaultFileMode,
	); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

func (f *FileSystem) saveWatchedFileInfosConcurrentSafe(watchedFiles watchedFilesMap) error {
	f.Lock()
	defer f.Unlock()

	return f.saveWatchedFileInfos(watchedFiles)
}

func (f *FileSystem) updateWatchedFileInfos(saveFunc func(watchedFilesMap) error) error {
	watchedFileInfos, err := f.loadWatchedFileInfos()
	if err != nil {
		f.logger.Warn().Err(err).Msg("Failed to load watched file infos.")

		watchedFileInfos = make(watchedFilesMap)
	}

	fileEntries, err := library.ListDirContent(f.solutionArchivesLocation, f.interestContentFilter)
	if err != nil {
		return errors.Stamp(err)
	}

	fileNames := library.ExtractNames(fileEntries)

	slices.Sort(fileNames)

	for fileName := range watchedFileInfos {
		if !slices.Contains(fileNames, fileName) {
			delete(watchedFileInfos, fileName)
		}
	}

	filteredEntries := make([]os.DirEntry, 0, len(fileEntries))

	for _, fileEntry := range fileEntries {
		fileInfo, ok := watchedFileInfos[fileEntry.Name()]
		if ok {
			entryInfo, err := fileEntry.Info()
			if err != nil {
				f.logger.Warn().Err(err).Msg("Failed to get file info.")

				continue
			}

			if entryInfo.ModTime().Equal(fileInfo.LastChangedAt) && entryInfo.Size() == fileInfo.Size {
				continue
			}
		}

		filteredEntries = append(filteredEntries, fileEntry)
	}

	newWatchedFileInfos := f.genWatchedFileInfos(filteredEntries)

	for fileName, watchedFileInfo := range newWatchedFileInfos {
		watchedFileInfos[fileName] = watchedFileInfo
	}

	if err := saveFunc(watchedFileInfos); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

func (f *FileSystem) watchFiles(filenameChan chan domain.FileEventDetails) {
	defer f.Done()

	for {
		select {
		case e, ok := <-f.watcher.Events:
			if !ok {
				f.logger.Warn().Msg("watcher events channel closed")
				return
			}
			log := f.logger.With().
				Str("source_event_type", e.Op.String()).
				Str("file_name", e.Name).
				Logger()

			log.Debug().Msg("event received")

			if err := f.updateWatchedFileInfos(f.saveWatchedFileInfosConcurrentSafe); err != nil {
				log.Error().Err(err).Msg("failed to update watched file infos")
			}

			var origin domain.FileOrigin
			if strings.HasPrefix(e.Name, f.solutionArchivesLocation) {
				origin = domain.SolutionArchivesOrigin
			} else if strings.HasPrefix(e.Name, f.solutionsLocation) {
				origin = domain.SolutionsOrigin
			}

			isDir, err := isDirectory(origin, e)
			if err != nil {
				if errors.Is(err, domain.ErrStorageProviderNotFound) {
					log.Debug().Err(err).Msg("object not found to determine if it is a directory")
					continue
				}
				log.Warn().Err(err).Msg("failed to determine if the object is a directory")
				continue
			}

			var objectNameVersion string
			switch origin {
			case domain.SolutionArchivesOrigin:
				if isDir {
					objectName, _ := filepath.Rel(f.solutionArchivesLocation, e.Name)
					objectNameVersion, _ = strings.CutPrefix(objectName, library.FileSystemBucketPrefix)
				} else {
					fileName := filepath.Base(e.Name)
					objectNameVersion = strings.TrimSuffix(fileName, ".iso")
				}
			case domain.SolutionsOrigin:
				objectNameVersion, _ = filepath.Rel(f.solutionsLocation, e.Name)
			}

			log.Debug().Msg("creating FileEventDetails")
			filenameChan <- domain.FileEventDetails{
				FullPathName: e.Name,
				ObjectName:   objectNameVersion,
				IsDir:        isDir,
				Origin:       origin,
				EventType:    e.Op.String(),
			}
			log.Debug().Msg("FileEventDetails created")
		case err, ok := <-f.watcher.Errors:
			if !ok {
				f.logger.Warn().Msg("watcher errors channel closed")

				return
			}

			f.logger.Error().Err(err).Msg("watcher error")
		}
	}
}

func (f *FileSystem) startWatchFiles(filenameChan chan domain.FileEventDetails) error {
	if err := f.updateWatchedFileInfos(f.saveWatchedFileInfos); err != nil {
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
			f.logger.Debug().Msg("creating an event for object " + dirEntry.Name())
			if !strings.HasPrefix(dirEntry.Name(), controlDir) {
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
			f.logger.Debug().Msg("creating an event for object " + dirEntry)
			f.watcher.Events <- fsnotify.Event{
				Name: dirEntry,
				Op:   fsnotify.Create,
			}
		}
	}(dirEntriesToAnalyze)
	return nil
}

func (f *FileSystem) stopWatchFiles() error {
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

func isDirectory(origin domain.FileOrigin, e fsnotify.Event) (bool, error) {
	switch e.Op {
	case fsnotify.Remove, fsnotify.Rename:
		if origin == domain.SolutionArchivesOrigin {
			fileName := filepath.Base(e.Name)
			if strings.HasPrefix(fileName, library.FileSystemBucketPrefix) {
				return true, nil
			} else if strings.HasSuffix(fileName, ".iso") {
				return false, nil
			}
			return false, errors.From(domain.ErrStorageProviderInternal).
				WithDetail("failed to determine if the object is a directory").
				WithProperty("object_name", e.Name).
				WithProperty("origin", origin).
				WithProperty("event_type", e.Op.String()).
				Throw()
		}
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
