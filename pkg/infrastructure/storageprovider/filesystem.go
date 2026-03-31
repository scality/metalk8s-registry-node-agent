package storageprovider

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strings"

	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"

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

func (f *FileSystem) SaveFile(
	fileName string,
	content io.Reader,
	perm os.FileMode,
) error {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return errors.Stamp(err)
	}

	if err := f.saveFile(fileName, content, perm); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

// ListFiles lists all the flat files in the root location of the storage
func (f *FileSystem) ListFiles() ([]string, error) {
	return f.listFiles()
}

func (f *FileSystem) GetFile(
	fileName string,
) (io.ReadCloser, error) {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return nil, errors.Stamp(err)
	}

	file, err := f.getFile(fileName)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return file, nil
}

// MoveFileToRoot moves a file from a bucket to the root location in the storage.
func (f *FileSystem) MoveFileToRoot(
	bucketName, fileName, newFileName string,
) error {
	if err := library.EnforceNamingConventions(newFileName); err != nil {
		return errors.Stamp(err)
	}

	if err := f.moveFileToRoot(bucketName, fileName, newFileName); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

// CreateMultipartFiles creates into a bucket: a metadata file, a multipart file recipient and a parts synthesis file.
func (f *FileSystem) CreateMultipartFiles(
	bucketName string,
	solutionArchive *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	if err := library.EnforceNamingConventions(solutionArchive.Name); err != nil {
		return nil, errors.Stamp(err)
	}
	meta, err := f.createMultipartFiles(bucketName, solutionArchive)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return meta, nil
}

// GetMultipartFile retrieves the multipart file recipient from a given bucket
// and returns its Solution Archive.
func (f *FileSystem) GetMultipartFile(
	bucketName string,
) (*domain.SolutionArchive, error) {
	meta, err := f.getMultipartFile(bucketName)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return meta, nil
}

// GetMultipartFileStatus retrieves the Solution Archive Status of a multipart file recipient
// based on bucketName and solutionArchiveMeta.
func (f *FileSystem) GetMultipartFileStatus(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	status, err := f.getSolutionArchiveStatus(bucketName, solutionArchiveMeta)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return status, nil
}

// DeleteMultipartFile deletes a multipart file recipient from a bucket based on bucketName and solutionArchiveMeta.
func (f *FileSystem) DeleteMultipartFile(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) error {
	if err := f.deleteMultipartFile(bucketName, solutionArchiveMeta); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

// WritePartToMultipartFile writes the content of the given part into the multipart file recipient
// on the bucket indicated by the given bucketName.
func (f *FileSystem) WritePartToMultipartFile(bucketName string,
	part *domain.Part,
) (*domain.SolutionArchiveStatus, error) {
	status, err := f.writePartToMultipartFile(bucketName, part)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return status, nil
}

// ConsolidateMultipartFile consolidates all the parts of a multipart file in a single flat file
// into the same bucket it is located.
func (f *FileSystem) ConsolidateMultipartFile(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
	perm os.FileMode,
) error {
	if err := f.consolidateMultipartFile(bucketName, solutionArchiveMeta, perm); err != nil {
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

// MountFile mounts the solution archive, identifed by its filename,
// into the solution location.
func (f *FileSystem) MountFile(
	fileName string,
	mountPoint string,
) error {
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
	err := f.watcher.Add(solutionPath)
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to add solution path to watcher").
			WithProperty("solution_path", solutionPath).
			Throw()
	}

	// Add a watcher for the mount path
	err = f.watcher.Add(mountPath)
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
				err := f.DeleteFile(fileName)
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

// AddWatchFileOrDirectory adds a file or directory to the watcher.
func (f *FileSystem) AddWatchFileOrDirectory(path string) error {
	return f.watcher.Add(path)
}

// RemoveWatchFileOrDirectory removes a file or directory from the watcher.
func (f *FileSystem) RemoveWatchFileOrDirectory(path string) error {
	return f.watcher.Remove(path)
}

// Bucket handling methods

func (f *FileSystem) genBucketPath(
	bucketName string,
) string {
	return filepath.Join(f.solutionArchivesLocation, library.FileSystemBucketPrefix+bucketName)
}

func (f *FileSystem) saveFile(
	fileName string,
	content io.Reader,
	perm os.FileMode,
) error {
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

func (f *FileSystem) listFiles() ([]string, error) {
	files, err := library.ListDirContentNames(f.solutionArchivesLocation, f.interestContentFilter)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return files, nil
}

func (f *FileSystem) getFile(
	fileName string,
) (io.ReadCloser, error) {
	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return nil, errors.Stamp(err)
	}

	file, err := library.GetFile(filePath)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return file, nil
}

func (f *FileSystem) genFileOnBucketPath(
	bucketName,
	fileName string,
) string {
	return filepath.Join(f.genBucketPath(bucketName), fileName)
}

func (f *FileSystem) moveFileToRoot(
	bucketName, fileName, newFileName string,
) error {
	bucketPath := f.genBucketPath(bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return errors.Stamp(err)
	}

	filePath := f.genFileOnBucketPath(bucketName, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return errors.Stamp(err)
	}

	newFilePath := filepath.Join(f.solutionArchivesLocation, newFileName)
	if err := os.Remove(newFilePath); err != nil && !os.IsNotExist(err) {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unexpected error while moving the file to the root location").
			WithProperty("current_file_path", filePath).
			WithProperty("new_file_path", newFilePath).
			WithProperty("while", "removing existing file from the root location").
			CausedBy(err).
			Throw()
	}

	if err := os.Rename(filePath, newFilePath); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unexpected error while moving the file to the root location").
			WithProperty("current_file_path", filePath).
			WithProperty("new_file_path", newFilePath).
			WithProperty("while", "moving the file from bucket to root location").
			CausedBy(err).
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

// filterOrphansMeta filters the orphaned SolutionArchiveMeta instances from the given
// solutionArchiveMetas list and returns the filtered list.
func (f *FileSystem) filterOrphansMeta(
	bucketPath string,
	solutionArchives []*domain.SolutionArchive,
) []*domain.SolutionArchive {
	filteredSolutionArchives := make([]*domain.SolutionArchive, 0, len(solutionArchives))

	for _, solutionArchive := range solutionArchives {
		if err := library.CheckFile(
			filepath.Join(
				bucketPath,
				solutionArchive.Name+library.FileSystemMultipartPartsSuffix,
			),
		); err != nil {
			f.logger.Warn().Err(err).Msg("The parts file is missing for the multipart file.")

			continue
		}

		if err := library.CheckFile(
			filepath.Join(
				bucketPath,
				solutionArchive.Name+library.FileSystemMultipartRecipientSuffix,
			),
		); err != nil {
			f.logger.Warn().Err(err).Msg("The recipient file is missing for the multipart file.")

			continue
		}

		filteredSolutionArchives = append(filteredSolutionArchives, solutionArchive)
	}

	return filteredSolutionArchives
}

// genBaseMultipartFilePath generates the base multipart file path for the
// given bucket name and file name without apply any suffix.
func (f *FileSystem) genBaseMultipartFilePath(bucketName, fileName string) string {
	return filepath.Join(f.genBucketPath(bucketName), fileName)
}

// genMultipartMetaFilePath generates the multipart meta file path for the given
// bucket name and file name.
func (f *FileSystem) genMultipartMetaFilePath(bucketName, fileName string) string {
	return f.genBaseMultipartFilePath(bucketName, fileName) + library.FileSystemMultipartMetaSuffix
}

// genMultipartPartsFilePath generates the multipart parts file path for the
// given bucket name and file name.
func (f *FileSystem) genMultipartPartsFilePath(bucketName, fileName string) string {
	return f.genBaseMultipartFilePath(bucketName, fileName) + library.FileSystemMultipartPartsSuffix
}

// genMultipartRecipientFilePath generates the multipart recipient file path for
// the given bucket name and file name.
func (f *FileSystem) genMultipartRecipientFilePath(bucketName, fileName string) string {
	return f.genBaseMultipartFilePath(bucketName, fileName) +
		library.FileSystemMultipartRecipientSuffix
}

// genMultipartFilePaths generates the multipart file paths for the given bucket
// name and file name.
func (f *FileSystem) genMultipartFilePaths(
	bucketName,
	fileName string,
) (metaFilePath, partsFilePath, recipientFilePath string) {
	return f.genMultipartMetaFilePath(bucketName, fileName),
		f.genMultipartPartsFilePath(bucketName, fileName),
		f.genMultipartRecipientFilePath(bucketName, fileName)
}

// loadSolutionArchiveMeta loads the Solution Archive Meta instance from
// the multipart meta file with the given file path.
func loadSolutionArchiveMeta(meta *domain.SolutionArchive, filePath string) error {
	metaFile, err := library.GetFile(filePath)
	if err != nil {
		return errors.Stamp(err)
	}

	defer metaFile.Close() // nolint: errcheck // No error check on defer.

	if err := json.NewDecoder(metaFile).Decode(meta); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to load the metadata of the multipart file").
			WithProperty("file_path", filePath).
			WithProperty("while", "decoding the metadata from json").
			CausedBy(err).
			Throw()
	}

	return nil
}

// getSolutionArchiveMetaList returns the list of Solution Archive Meta instances
// in the bucket with the given bucketPath.
func (f *FileSystem) getSolutionArchiveMeta(bucketPath string) (*domain.SolutionArchive, error) {
	metaFileNames, err := library.ListDirContentNames(bucketPath, library.MultipartMetaFilter)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	solutionArchiveMetas := make([]*domain.SolutionArchive, 0, len(metaFileNames))

	for _, metaFileName := range metaFileNames {
		var solutionArchiveMeta domain.SolutionArchive

		if err := loadSolutionArchiveMeta(&solutionArchiveMeta, filepath.Join(bucketPath, metaFileName)); err != nil {
			return nil, errors.Stamp(err)
		}

		solutionArchiveMetas = append(solutionArchiveMetas, &solutionArchiveMeta)
	}

	solutionArchiveMetas = f.filterOrphansMeta(bucketPath, solutionArchiveMetas)
	if len(solutionArchiveMetas) < 1 {
		return nil, errors.From(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			WithDetail("no multipart files found in the bucket").
			WithProperty("bucket_name", bucketPath).
			Throw()
	}
	if len(solutionArchiveMetas) > 1 {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("too many multipart files found in the bucket").
			WithProperty("bucket_name", bucketPath).
			Throw()
	}

	return solutionArchiveMetas[0], nil
}

func (f *FileSystem) createMultipartFiles(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	if err := library.CheckDir(f.genBucketPath(bucketName)); err != nil {
		return nil, errors.Stamp(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		solutionArchiveMeta.Name,
	)

	err := library.CheckFile(metaFilePath)
	if err == nil {
		solutionArchiveStatus, err := f.getSolutionArchiveStatus(bucketName, solutionArchiveMeta)
		if err != nil {
			return nil, errors.Stamp(err)
		}

		return solutionArchiveStatus, nil
	}

	if !errors.Is(err,
		errors.Intercept(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			Throw()) {
		return nil, errors.Stamp(err)
	}

	metaContentBytes, err := json.Marshal(solutionArchiveMeta)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to save the solution archive metadata").
			WithProperty("solution_archive", solutionArchiveMeta.Name).
			WithProperty("version", solutionArchiveMeta.Version).
			WithProperty("while", "marshalling the metadata to json format").
			Throw()
	}

	cleanUp := func() {
		os.Remove(metaFilePath)      // nolint: errcheck // No *PathError error possible.
		os.Remove(partsFilePath)     // nolint: errcheck // No *PathError error possible.
		os.Remove(recipientFilePath) // nolint: errcheck // No *PathError error possible.
	}

	if err := library.SaveFile(
		metaFilePath,
		bytes.NewReader(metaContentBytes),
		library.FileSystemDefaultFileMode,
	); err != nil {
		cleanUp()

		return nil, errors.Stamp(err)
	}

	if err := library.CreateEmptyFile(
		partsFilePath,
		0,
		library.FileSystemDefaultFileMode,
	); err != nil {
		cleanUp()

		return nil, errors.Stamp(err)
	}

	if err := library.CreateEmptyFile(
		recipientFilePath,
		solutionArchiveMeta.Size,
		library.FileSystemDefaultFileMode,
	); err != nil {
		cleanUp()

		return nil, errors.Stamp(err)
	}

	return &domain.SolutionArchiveStatus{
		SolutionArchive: solutionArchiveMeta,
		Parts:           make(map[int64]*domain.PartMeta),
	}, nil
}

func (f *FileSystem) getMultipartFile(bucketName string) (*domain.SolutionArchive, error) {
	bucketPath := f.genBucketPath(bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return nil, errors.Stamp(err)
	}

	solutionArchiveMeta, err := f.getSolutionArchiveMeta(bucketPath)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return solutionArchiveMeta, nil
}

func (f *FileSystem) getSolutionArchiveStatus(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	if err := library.CheckDir(f.genBucketPath(bucketName)); err != nil {
		return nil, errors.Stamp(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		solutionArchiveMeta.Name,
	)

	if err := library.CheckFile(metaFilePath); err != nil {
		return nil, errors.Stamp(err)
	}

	if err := library.CheckFile(partsFilePath); err != nil {
		return nil, errors.Stamp(err)
	}

	if err := library.CheckFile(recipientFilePath); err != nil {
		return nil, errors.Stamp(err)
	}

	metaFile, err := library.GetFile(metaFilePath)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	defer metaFile.Close() // nolint: errcheck // No error check on defer.

	partsFile, err := library.GetFile(partsFilePath)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	defer partsFile.Close() // nolint: errcheck // No error check on defer.

	var storedSolutionArchiveMeta domain.SolutionArchive
	if err := json.NewDecoder(metaFile).Decode(&storedSolutionArchiveMeta); err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to load the metadata of the multipart file").
			WithProperty("file_path", metaFilePath).
			WithProperty("while", "decoding the metadata from json").
			CausedBy(err).
			Throw()
	}

	if err := library.CompareSolutionArchiveMetas(solutionArchiveMeta, &storedSolutionArchiveMeta); err != nil {
		return nil, errors.Stamp(err)
	}

	partMetas := make(map[int64]*domain.PartMeta)
	for {
		var partMeta domain.PartMeta

		err = binary.Read(partsFile, binary.LittleEndian, &partMeta)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, errors.From(domain.ErrStorageProviderInternal).
				WithIdentifier(500000).
				WithDetail("unable to load the parts metadata of the multipart file").
				WithProperty("file_path", partsFilePath).
				WithProperty("while", "decoding the parts metadata from binary").
				CausedBy(err).
				Throw()
		}

		partMetas[partMeta.Start] = &partMeta
	}

	return &domain.SolutionArchiveStatus{
		SolutionArchive: solutionArchiveMeta,
		Parts:           partMetas,
	}, nil
}

func (f *FileSystem) deleteMultipartFile(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) error {
	if err := library.CheckDir(f.genBucketPath(bucketName)); err != nil {
		return errors.Stamp(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		solutionArchiveMeta.Name,
	)

	if err := library.CheckFile(metaFilePath); err != nil {
		return errors.Stamp(err)
	}

	if err := library.CheckFile(partsFilePath); err != nil {
		return errors.Stamp(err)
	}

	if err := library.CheckFile(recipientFilePath); err != nil {
		return errors.Stamp(err)
	}

	problems := make(map[string]any)

	if err := os.Remove(metaFilePath); err != nil {
		problems["problem_remove_meta_file"] = err
	}

	if err := os.Remove(partsFilePath); err != nil {
		problems["problem_remove_parts_file"] = err
	}

	if err := os.Remove(recipientFilePath); err != nil {
		problems["problem_remove_recipient_filer"] = err
	}

	if len(problems) > 0 {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to delete the multipart file").
			WithProperties(problems).
			Throw()
	}

	return nil
}

func (f *FileSystem) writePartToMultipartFile(
	bucketName string,
	part *domain.Part,
) (*domain.SolutionArchiveStatus, error) {
	solutionArchiveStatus, err := f.getSolutionArchiveStatus(bucketName, part.SolutionArchive)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	partsFilePath := f.genMultipartPartsFilePath(bucketName, part.SolutionArchive.Name)
	recipientFilePath := f.genMultipartRecipientFilePath(bucketName, part.SolutionArchive.Name)

	partsFile, err := os.OpenFile(partsFilePath, os.O_WRONLY, library.FileSystemDefaultFileMode)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to open the parts file").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	defer partsFile.Close() // nolint: errcheck // No error check on defer.

	recipientFile, err := os.OpenFile(recipientFilePath, os.O_RDWR, library.FileSystemDefaultFileMode)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to open the recipient file").
			WithProperty("file_path", recipientFilePath).
			CausedBy(err).
			Throw()
	}

	defer recipientFile.Close() // nolint: errcheck // No error check on defer.

	_, err = recipientFile.Seek(part.Meta.Start, io.SeekStart)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to move to the start of the part in the recipient file").
			WithProperty("file_path", recipientFilePath).
			WithProperty("part_start", part.Meta.Start).
			CausedBy(err).
			Throw()
	}

	written, err := io.Copy(recipientFile, part.Content)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to write the part to the recipient file").
			WithProperty("file_path", recipientFilePath).
			WithProperty("part_size", part.Meta.Size()).
			CausedBy(err).
			Throw()
	}

	if written != part.Meta.Size() {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("the part was not fully written to the recipient file").
			WithProperty("file_path", recipientFilePath).
			WithProperty("part_size", part.Meta.Size()).
			WithProperty("written", written).
			Throw()
	}

	_, err = partsFile.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to move to the end of the parts file").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	err = binary.Write(partsFile, binary.LittleEndian, part.Meta)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to write the part metadata to the parts file").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	solutionArchiveStatus.Parts[part.Meta.Start] = part.Meta

	return solutionArchiveStatus, nil
}

func (f *FileSystem) consolidateMultipartFile(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
	perm os.FileMode,
) error {
	solutionArchiveStatus, err := f.getSolutionArchiveStatus(bucketName, solutionArchiveMeta)
	if err != nil {
		return err
	}

	if !solutionArchiveStatus.IsComplete() {
		return errors.From(domain.ErrStorageProviderBusinessRuleViolation).
			WithIdentifier(422001).
			WithDetail("unable to consolidate multipart file because it is not complete").
			WithProperty("bucket_name", bucketName).
			WithProperty("solution_archive_name", solutionArchiveMeta.Name).
			Throw()
	}

	baseFilePath := f.genBaseMultipartFilePath(bucketName, solutionArchiveMeta.Name)
	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		solutionArchiveMeta.Name,
	)

	// Calculate the SHA256 hash of the recipient file
	recipientFile, err := library.GetFile(recipientFilePath)
	if err != nil {
		return errors.Stamp(err)
	}

	defer recipientFile.Close() // nolint: errcheck // No error check on defer.

	hasher := sha256.New()
	if _, err := io.Copy(hasher, recipientFile); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to calculate the hash of the recipient file").
			WithProperty("file_path", recipientFilePath).
			CausedBy(err).
			Throw()
	}

	calculedHash := hex.EncodeToString(hasher.Sum(nil))

	if calculedHash != solutionArchiveMeta.Hash {
		return errors.From(domain.ErrStorageProviderBusinessRuleViolation).
			WithIdentifier(422001).
			WithDetail("the hash of the recipient file does not match the solution archive metadata").
			WithProperty("component", solutionArchiveMeta.Name).
			WithProperty("version", solutionArchiveMeta.Version).
			WithProperty("expected_hash", solutionArchiveMeta.Hash).
			WithProperty("calculed_hash", calculedHash).
			Throw()
	}

	if err := os.Rename(recipientFilePath, baseFilePath); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to rename the recipient file").
			WithProperty("from", recipientFilePath).
			WithProperty("to", baseFilePath).
			CausedBy(err).
			Throw()
	}

	if err := os.Chmod(baseFilePath, perm); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to change the permissions of the recipient file").
			WithProperty("file_path", baseFilePath).
			WithProperty("permissions", perm).
			CausedBy(err).
			Throw()
	}

	// Remove other files
	problems := make(map[string]any)

	if err := os.Remove(metaFilePath); err != nil {
		problems["problem_remove_meta_file"] = err
	}

	if err := os.Remove(partsFilePath); err != nil {
		problems["problem_remove_parts_file"] = err
	}

	if len(problems) > 0 {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("unable to clean up multipart file bundle").
			WithProperties(problems).
			Throw()
	}

	return nil
}

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
					log.Debug().Err(err).Msg("object not found to determine if it is a directory")
					continue
				}
				log.Warn().Err(err).Msg("failed to determine if the object is a directory")
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

func (f *FileSystem) StartWatchFiles(filenameChan chan domain.FileEventDetails) error {
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

func (f *FileSystem) CleanUnusedSolutions(path string, isDir bool) error {
	f.Lock()
	defer f.Unlock()

	if isDir {
		// Unmount the solution
		err := f.UnmountFile(path)
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
			return errors.From(domain.ErrSolutionArchiveCleanerInternal).
				CausedBy(err).
				WithDetail("failed to delete unused solution directory").
				WithProperty("path", path).
				Throw()
		}

		f.logger.Debug().
			Str("path", path).
			Msg("finished to clean unused solution directory")

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

	f.logger.Debug().
		Str("path", path).
		Msg("finished to clean unused solution")

	return nil
}

func (f *FileSystem) CleanUnusedSolutionArchives(path string, isDir bool) error {
	f.Lock()
	defer f.Unlock()

	if isDir {
		err := os.RemoveAll(path)
		if err != nil {
			return errors.From(domain.ErrSolutionArchiveCleanerInternal).
				CausedBy(err).
				WithDetail("failed to delete unused solution archive directory").
				WithProperty("path", path).
				Throw()
		}

		f.logger.Debug().
			Str("path", path).
			Msg("finished to clean unused solution archive directory")

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

	f.logger.Debug().
		Str("path", path).
		Msg("finished to clean unused solution archive")

	return nil
}
