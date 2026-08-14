package storageprovider

import (
	"context"
	"io"
	"log/slog"
	"os"
	"regexp"
	"sync"

	"github.com/fsnotify/fsnotify"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type (
	MockFileSystem struct {
		sync.RWMutex
		sync.WaitGroup

		logger *slog.Logger

		solutionArchivesLocation string
		solutionsLocation        string
		interestContentFilter    library.ContentFilter
		watcher                  *fsnotify.Watcher
		files                    []string
		buckets                  []string
		multipartMeta            map[string]*domain.SolutionArchive
	}

	MockFileOpts struct {
		Logger                     *slog.Logger
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
var _ service.MultipartRemover = &MockFileSystem{}
var _ service.MultipartStorer = &MockFileSystem{}

func NewMockFileSystem(opts *MockFileOpts) *MockFileSystem {
	return &MockFileSystem{
		logger:                   opts.Logger,
		solutionArchivesLocation: opts.SolutionArchivesLocation,
		solutionsLocation:        opts.SolutionsLocation,
		interestContentFilter:    library.NewRegexNormalFileFilter(opts.InterestContentFilterRegex),
		files:                    []string{},
		buckets:                  []string{},
		multipartMeta:            make(map[string]*domain.SolutionArchive),
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
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(81),
			errors.WithDetail("failed to create watcher"),
			errors.CausedBy(err),
		)
	}

	f.watcher = watcher

	return nil
}

// ListFiles lists all the flat files in the root location of the storage
func (f *MockFileSystem) ListFiles() ([]string, error) {
	return f.files, nil
}

func (f *MockFileSystem) GetFile(
	fileName string,
) (io.ReadCloser, error) {
	return io.ReadCloser(nil), nil
}

func (f *MockFileSystem) GetPart(
	fileName string,
	start int64,
	end int64,
) (io.ReadCloser, error) {
	return io.ReadCloser(nil), nil
}

func (f *MockFileSystem) DeleteFile(
	fileName string,
) error {
	return nil
}

func (f *MockFileSystem) CreateBucket(
	bucketName string,
) error {
	if bucketName == "solution-4-4.2.8" {
		return errors.Wrap(domain.ErrStorageProviderInternal,
			errors.WithIdentifier(120),
			errors.WithDetail("error creating bucket"),
			errors.WithProperty("bucket_name", bucketName),
		)
	}
	f.buckets = append(f.buckets, bucketName)
	return nil
}

// ListBuckets lists all the buckets in the storage and returns their bucketNames,
// after having removed FileSystemBucketPrefix.
func (f *MockFileSystem) ListBuckets() ([]string, error) {
	return f.buckets, nil
}

func (f *MockFileSystem) DeleteBucket(
	bucketName string,
) error {
	for i, b := range f.buckets {
		if b == bucketName {
			f.buckets = append(f.buckets[:i], f.buckets[i+1:]...)
			break
		}
	}
	delete(f.multipartMeta, bucketName)
	return nil
}

func (f *MockFileSystem) Consolidate(
	_ context.Context,
	sessionBucket string,
	solutionArchiveFromManifest *domain.SolutionArchive,
	perm os.FileMode,
) error {
	solutionArchiveFileName := library.GenSolutionArchiveFileName(solutionArchiveFromManifest)
	f.files = append(f.files, solutionArchiveFileName)
	return nil
}

func (f *MockFileSystem) CreateMultipartFiles(
	_ context.Context,
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	f.multipartMeta[bucketName] = solutionArchiveMeta
	return &domain.SolutionArchiveStatus{
		SolutionArchive: solutionArchiveMeta,
		Parts:           make(map[int64]*domain.PartMeta),
	}, nil
}

func (f *MockFileSystem) GetMultipartFile(
	_ context.Context,
	bucketName string,
) (*domain.SolutionArchive, error) {
	if meta, ok := f.multipartMeta[bucketName]; ok {
		return meta, nil
	}
	return &domain.SolutionArchive{}, nil
}

func (f *MockFileSystem) GetMultipartFileStatus(
	_ context.Context,
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) (*domain.SolutionArchiveStatus, error) {
	return &domain.SolutionArchiveStatus{}, nil
}

func (f *MockFileSystem) DeleteMultipartFile(
	bucketName string,
	solutionArchiveMeta *domain.SolutionArchive,
) error {
	return nil
}

func (f *MockFileSystem) StorePart(
	_ context.Context,
	bucketName string,
	solutionArchiveFromManifest *domain.SolutionArchive,
	part *domain.Part,
) (*domain.SolutionArchiveStatus, error) {
	// Drain the part body so any wrapping reader (e.g. the external downloader's
	// digest-verifying tee reader) observes the full payload. Without this the
	// trailer-based Content-Digest check on Close() always sees an empty hash.
	if part.Content != nil {
		if _, err := io.Copy(io.Discard, part.Content); err != nil {
			return nil, errors.Wrap(domain.ErrStorageProviderInternal,
				errors.WithIdentifier(175),
				errors.WithDetail("unable to write the part content"),
				errors.CausedBy(err),
			)
		}
	}

	if solutionArchiveFromManifest.Size == 0 {
		solutionArchiveFromManifest.Size = part.SolutionArchive.Size
		f.multipartMeta[bucketName] = solutionArchiveFromManifest
	}
	sa := f.multipartMeta[bucketName]
	return &domain.SolutionArchiveStatus{
		SolutionArchive: sa,
		Parts: map[int64]*domain.PartMeta{
			part.Meta.Start: {Start: part.Meta.Start, End: part.Meta.End},
		},
	}, nil
}

func (f *MockFileSystem) CommitPart(_ context.Context, bucketName string, part *domain.Part) error {
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

	return "", errors.Wrap(domain.ErrStorageProviderNotFound,
		errors.WithIdentifier(77),
		errors.WithDetail("file not found"),
		errors.WithProperty("file_name", filename),
	)
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

func (f *MockFileSystem) CleanUnusedSolutionArchives(_ context.Context, path string, isDir bool) error {
	return nil
}

func (f *MockFileSystem) CleanUnusedSolutions(_ context.Context, path string, isDir bool) error {
	return nil
}
