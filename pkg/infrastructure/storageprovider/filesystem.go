package storageprovider

import (
	"bytes"
	"encoding/json"

	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type (
	FileSystem struct {
		sync.RWMutex
		sync.WaitGroup

		logger *zerolog.Logger

		solutionArchivesLocation string
		solutionsLocation        string
		interestContentFilter    library.ContentFilter
		watchedFileInfos         watchedFilesMap
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

var _ service.StorageProvider = &FileSystem{}

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
	}
}

func (f *FileSystem) ControlDir() string {
	return controlDir
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

	return nil
}

// SPEC: The multipart file recipient is composed by three files: the
// solution Archive metadata, the part index and the recipient

// SPEC: The metadata file has the same name as the final solution Archive but
// suffixed with ".meta"

// SPEC: The metadata file stores the Solution Archive structure encoded as JSON

// SPEC: The part index has the same name as the final solution Archive but suffixed
// with ".parts"

// SPEC: The part index must be created with an empty content

// SPEC: The part index stores the PartMeta structures as binary
// (pkg.go.dev/encoding/binary)

// SPEC: The recipient has the same name as the final solution Archive but suffixed
// with ".recipient"

// ===================================== TO REFACTOR ========================================= //

type (
	watchedFileInfo struct {
		Size          int64     `json:"size"`
		LastChangedAt time.Time `json:"last_changed_at"`
		Hash          string    `json:"hash"`
	}

	watchedFilesMap map[string]*watchedFileInfo
)

const watchedFilesInfoName = "watched_files_info.json"

func (f *FileSystem) genControlDirPath() string {
	return filepath.Join(f.solutionArchivesLocation, controlDir)
}

func (f *FileSystem) genWatchedFilesPath() string {
	return filepath.Join(f.genControlDirPath(), watchedFilesInfoName)
}

func (f *FileSystem) genWatchedFileInfo(fileEntry os.DirEntry) (*watchedFileInfo, error) {
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

	return &watchedFileInfo{
		Size:          fileInfo.Size(),
		LastChangedAt: fileInfo.ModTime(),
		Hash:          hash,
	}, nil
}

func (f *FileSystem) genWatchedFileInfos(fileEntries []os.DirEntry) watchedFilesMap {
	type watchedFileEntry struct {
		fileName string
		fileInfo *watchedFileInfo
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
	watchedFileMap := make(map[string]*watchedFileInfo)
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
	f.watchedFileInfos = watchedFileInfos

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

func (f *FileSystem) InitWatchedFileInfos() error {
	return f.updateWatchedFileInfos(f.saveWatchedFileInfos)
}

func (f *FileSystem) RefreshWatchedFileInfos() error {
	return f.updateWatchedFileInfos(f.saveWatchedFileInfosConcurrentSafe)
}

func (f *FileSystem) updateWatchedFileInfos(saveFunc func(watchedFilesMap) error) error {
	// Load current stored watched file infos.
	watchedFileInfos, err := f.loadWatchedFileInfos()
	if err != nil {
		f.logger.Warn().Err(err).Msg("Failed to load watched file infos.")

		watchedFileInfos = make(watchedFilesMap)
	}

	// List actual interest content.
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

	// Filter out from fileEntries those which are up to date.
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

	// Generate watchedFileInfo for the new interesting content files.
	newWatchedFileInfos := f.genWatchedFileInfos(filteredEntries)

	// Merge new watched file infos with the old ones.
	for fileName, watchedFileInfo := range newWatchedFileInfos {
		watchedFileInfos[fileName] = watchedFileInfo
	}

	// Save watched file infos to disk.
	if err := saveFunc(watchedFileInfos); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

// GetArchiveHash retrieves the hash of a file from the storage backend.
func (f *FileSystem) GetArchiveHash(filename string) (string, error) {
	if _, ok := f.watchedFileInfos[filename]; !ok {
		return "", errors.From(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			WithDetail("file not found").
			WithProperty("file_name", filename).
			Throw()
	}

	return f.watchedFileInfos[filename].Hash, nil
}

// GetArchiveSize retrieves the size of a file from the storage backend.
func (f *FileSystem) GetArchiveSize(filename string) (int64, error) {
	if _, ok := f.watchedFileInfos[filename]; !ok {
		return 0, errors.From(domain.ErrStorageProviderNotFound).
			WithDetail("file not found").
			WithProperty("file_name", filename).
			Throw()
	}

	return f.watchedFileInfos[filename].Size, nil
}
