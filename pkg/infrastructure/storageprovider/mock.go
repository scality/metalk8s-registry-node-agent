package storageprovider

import (
	"io"
	"os"
	"regexp"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type (
	MockFileSystem struct {
		sync.RWMutex
		sync.WaitGroup

		logger *zerolog.Logger

		solutionArchivesLocation string
		solutionsLocation        string
		interestContentFilter    library.ContentFilter
		watcher                  *fsnotify.Watcher
		files                    []string
	}

	MockFileOpts struct {
		Logger                     *zerolog.Logger
		SolutionArchivesLocation   string
		SolutionsLocation          string
		InterestContentFilterRegex *regexp.Regexp
	}
)

var _ service.StorageProvider = &MockFileSystem{}
var _ service.BucketManager = &MockFileSystem{}
var _ service.ArchiveRemover = &MockFileSystem{}
var _ service.ArchiveReader = &MockFileSystem{}
var _ service.FileWatcher = &MockFileSystem{}
var _ service.ArchiveMounter = &MockFileSystem{}
var _ service.ArchiveCleaner = &MockFileSystem{}
var _ service.ArchiveLister = &MockFileSystem{}
var _ service.MultipartInspector = &MockFileSystem{}

func NewMockFileSystem(opts *MockFileOpts) *MockFileSystem {
	return &MockFileSystem{
		logger:                   opts.Logger,
		solutionArchivesLocation: opts.SolutionArchivesLocation,
		solutionsLocation:        opts.SolutionsLocation,
		interestContentFilter:    library.NewRegexNormalFileFilter(opts.InterestContentFilterRegex),
		files:                    []string{},
	}
}

// ControlDir returns the control directory.
func (f *MockFileSystem) ControlDir() string {
	return controlDir
}

// InitWatchedFileInfos initializes the watched file infos.
func (f *MockFileSystem) InitWatchedFileInfos() error {
	return nil
}

// RefreshWatchedFileInfos refreshes the watched file infos.
func (f *MockFileSystem) RefreshWatchedFileInfos() error {
	return nil
}

func (f *MockFileSystem) Init() error {
	f.Lock()
	defer f.Unlock()

	// For testing purposes, consider existing following ISO files
	f.files = append(f.files, "solution-2-4.2.1.iso")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("failed to create watcher").
			Throw()
	}

	f.watcher = watcher

	return nil
}

func (f *MockFileSystem) SaveFile(
	fileName string,
	content io.Reader,
	perm os.FileMode,
) error {
	f.files = append(f.files, fileName)
	return nil
}

// ListFiles lists all the flat files in the root location of the storage
func (f *MockFileSystem) ListFiles() ([]string, error) {
	return f.listFiles()
}

func (f *MockFileSystem) GetFile(
	fileName string,
) (io.ReadCloser, error) {
	return io.ReadCloser(nil), nil
}

func (f *MockFileSystem) DeleteFile(
	fileName string,
) error {
	if err := f.deleteFile(fileName); err != nil {
		return errors.Stamp(err)
	}
	return nil
}

func (f *MockFileSystem) HashFile(
	fileName string,
) (string, error) {
	hash, err := f.hashFile(fileName)
	if err != nil {
		return "", errors.Stamp(err)
	}

	return hash, nil
}

func (f *MockFileSystem) CreateBucket(
	bucketName string,
) error {
	return f.createBucket(bucketName)
}

// ListBuckets lists all the buckets in the storage and returns their bucketNames,
// after having removed FileSystemBucketPrefix.
func (f *MockFileSystem) ListBuckets() ([]string, error) {
	return f.listBuckets()
}

func (f *MockFileSystem) DeleteBucket(
	bucketName string,
) error {
	if err := f.deleteBucket(bucketName); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

func (f *MockFileSystem) MoveFileToRoot(
	bucketName, fileName, newFileName string,
) error {
	if err := f.moveFileToRoot(bucketName, fileName, newFileName); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

func (f *MockFileSystem) CreateMultipartFiles(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	meta, err := f.createMultipartFiles(bucketName, solutionArchiveMeta)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return meta, nil
}

func (f *MockFileSystem) GetMultipartFile(
	bucketName string,
) (*domain.SolutionArchive, error) {
	meta, err := f.getMultipartFile(bucketName)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return meta, nil
}

func (f *MockFileSystem) GetMultipartFileStatus(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	status, err := f.getSolutionArchiveStatus(bucketName, solutionArchiveMeta)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return status, nil
}

func (f *MockFileSystem) DeleteMultipartFile(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) error {
	if err := f.deleteMultipartFile(bucketName, solutionArchiveMeta); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

func (f *MockFileSystem) WritePartToMultipartFile(bucketName string,
	part *domain.Part,
) (*domain.SolutionArchiveStatus, error) {
	status, err := f.writePartToMultipartFile(bucketName, part)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return status, nil
}

func (f *MockFileSystem) ConsolidateMultipartFile(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
	perm os.FileMode,
) error {
	if err := f.consolidateMultipartFile(bucketName, solutionArchiveMeta, perm); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

func (f *MockFileSystem) GetArchiveHash(filename string) (string, error) {
	hashMap := map[string]string{
		"solution-2-4.2.1.iso": "ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26",
		"solution-3-4.2.1.iso": "95162a9fe88f9d11c7f7ef7dc20c2426814e188fd858b27adb5274d3689675af",
	}

	if hash, ok := hashMap[filename]; ok {
		return hash, nil
	}

	return "", errors.From(domain.ErrStorageProviderNotFound).
		WithIdentifier(404000).
		WithDetail("file not found").
		WithProperty("file_name", filename).
		Throw()
}

func (f *MockFileSystem) GetArchiveSize(filename string) (int64, error) {
	return 0, nil
}

func (f *MockFileSystem) MountFile(
	fileName string,
	mountPoint string,
) error {
	return nil
}

func (f *MockFileSystem) UnmountFile(
	mountPoint string,
) error {
	return nil
}

// Bucket handling methods

func (f *MockFileSystem) createBucket(
	_ string,
) error {
	return nil
}

func (f *MockFileSystem) listBuckets() ([]string, error) {
	return []string{}, nil
}

func (f *MockFileSystem) deleteBucket(
	_ string,
) error {
	return nil
}

func (f *MockFileSystem) listFiles() ([]string, error) {
	return f.files, nil
}

func (f *MockFileSystem) deleteFile(
	_ string,
) error {
	return nil
}

func (f *MockFileSystem) hashFile(
	fileName string,
) (string, error) {
	return f.GetArchiveHash(fileName)
}

func (f *MockFileSystem) moveFileToRoot(_, _, _ string) error {
	return nil
}

// nolint:unparam
func (f *MockFileSystem) createMultipartFiles(
	_ string,
	solutionArchiveMeta *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	return &domain.SolutionArchiveStatus{
		SolutionArchive: solutionArchiveMeta,
		Parts:           make(map[int64]*domain.PartMeta),
	}, nil
}

func (f *MockFileSystem) getMultipartFile(_ string) (*domain.SolutionArchive, error) {
	return &domain.SolutionArchive{}, nil
}

func (f *MockFileSystem) getSolutionArchiveStatus(
	_ string,
	_ *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	return &domain.SolutionArchiveStatus{}, nil
}

func (f *MockFileSystem) deleteMultipartFile(_ string, _ *domain.SolutionArchive) error {
	return nil
}

func (f *MockFileSystem) writePartToMultipartFile(
	_ string,
	_ *domain.Part,
) (*domain.SolutionArchiveStatus, error) {
	return &domain.SolutionArchiveStatus{}, nil
}

func (f *MockFileSystem) consolidateMultipartFile(
	_ string,
	_ *domain.SolutionArchive,
	_ os.FileMode,
) error {
	return nil
}

func (f *MockFileSystem) StartWatchFiles(_ chan domain.FileEventDetails) error {
	return nil
}

// nolint:errcheck
func (f *MockFileSystem) StopWatchFiles() error {
	defer f.Wait()
	f.watcher.Close()
	return nil
}

func (f *MockFileSystem) AddWatchFileOrDirectory(path string) error {
	return nil
}

func (f *MockFileSystem) RemoveWatchFileOrDirectory(path string) error {
	return nil
}

func (f *MockFileSystem) CleanUnusedSolutionArchives(path string, isDir bool) error {
	return nil
}

func (f *MockFileSystem) CleanUnusedSolutions(path string, isDir bool) error {
	return nil
}
