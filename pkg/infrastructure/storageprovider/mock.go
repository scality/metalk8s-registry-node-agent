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

		solutionArchiveLocation string
		interestContentFilter   library.ContentFilter
		watcher                 *fsnotify.Watcher
	}

	MockFileOpts struct {
		Logger                     *zerolog.Logger
		SolutionArchiveLocation    string
		InterestContentFilterRegex *regexp.Regexp
	}
)

var _ service.StorageProvider = &MockFileSystem{}

func NewMockFileSystem(opts *MockFileOpts) *MockFileSystem {
	return &MockFileSystem{
		logger:                  opts.Logger,
		solutionArchiveLocation: opts.SolutionArchiveLocation,
		interestContentFilter:   library.NewRegexNormalFileFilter(opts.InterestContentFilterRegex),
	}
}

func (f *MockFileSystem) Init() error {
	f.Lock()
	defer f.Unlock()

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

func (f *MockFileSystem) Start(filenameCh chan string) error {
	f.Lock()
	defer f.Unlock()

	return f.startWatchFiles(filenameCh)
}

func (f *MockFileSystem) Stop() error {
	return f.stopWatchFiles()
}

func (f *MockFileSystem) SaveFile(
	fileName string,
	content io.Reader,
	perm os.FileMode,
) error {
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

func (f *MockFileSystem) GetHashFromFileInfos(filename string) (string, error) {
	hashMap := map[string]string{
		"solution-2-4.2.1.iso": "ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26",
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

func (f *MockFileSystem) GetSizeFromFileInfos(filename string) (int64, error) {
	return 0, nil
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
	// For testing purposes, consider existing following ISO files
	isoFiles := []string{
		"solution-2-4.2.1.iso",
	}
	return isoFiles, nil
}

func (f *MockFileSystem) deleteFile(
	_ string,
) error {
	return nil
}

func (f *MockFileSystem) hashFile(
	fileName string,
) (string, error) {
	return f.GetHashFromFileInfos(fileName)
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

func (f *MockFileSystem) startWatchFiles(_ chan string) error {
	return nil
}

// nolint:errcheck
func (f *MockFileSystem) stopWatchFiles() error {
	defer f.Wait()
	f.watcher.Close()
	return nil
}
