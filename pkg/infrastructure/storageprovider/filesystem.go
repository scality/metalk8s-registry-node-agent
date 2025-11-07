package storageprovider

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
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

		artifactLocation      string
		interestContentFilter library.ContentFilter
		watchedFileInfos      watchedFilesMap
		watcher               *fsnotify.Watcher
	}

	FileOpts struct {
		Logger                     *zerolog.Logger
		ArtifactLocation           string
		InterestContentFilterRegex *regexp.Regexp
	}
)

const (
	controlDir = ".storageprovider"
)

var _ service.StorageProvider = &FileSystem{}

func NewFileSystem(opts *FileOpts) *FileSystem {
	return &FileSystem{
		logger:                opts.Logger,
		artifactLocation:      opts.ArtifactLocation,
		interestContentFilter: library.NewRegexNormalFileFilter(opts.InterestContentFilterRegex),
	}
}

// Init initializes the storage provider.
func (f *FileSystem) Init() error {
	f.Lock()
	defer f.Unlock()

	if err := os.MkdirAll(f.artifactLocation, library.FileSystemDefaultDirMode); err != nil {
		return errors.From(domain.ErrStorageProviderInit).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("Failed to create artifact location.").
			WithProperty("artifact_location", f.artifactLocation).
			Throw()
	}

	controlDirectoryPath := filepath.Join(f.artifactLocation, controlDir)

	if err := os.MkdirAll(controlDirectoryPath, library.FileSystemDefaultDirMode); err != nil {
		return errors.From(domain.ErrStorageProviderInit).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("Failed to create control directory.").
			WithProperty("control_directory", controlDirectoryPath).
			Throw()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.From(domain.ErrStorageProviderInit).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("Failed to create watcher.").
			Throw()
	}

	f.watcher = watcher

	return nil
}

// Start starts the watcher on the storage provider.
func (f *FileSystem) Start(filenameCh chan string) error {
	f.Lock()
	defer f.Unlock()

	return f.startWatchFiles(filenameCh)
}

// Stop stops the watcher on the storage provider.
func (f *FileSystem) Stop() error {
	return f.stopWatchFiles()
}

// ListFiles lists all the flat files in the root location of the storage
func (f *FileSystem) ListFiles() ([]string, error) {
	return f.listFiles()
}

// DeleteFile deletes a file from the root location in the storage based on its fileName.
func (f *FileSystem) DeleteFile(
	fileName string,
) error {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return errors.Stamp(err)
	}

	if err := f.deleteFile(fileName); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

// HashFile calculates the hash of a file from the root location in the storage based on its fileName.
func (f *FileSystem) HashFile(
	fileName string,
) (string, error) {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return "", errors.Stamp(err)
	}

	hash, err := f.hashFile(fileName)
	if err != nil {
		return "", errors.Stamp(err)
	}

	return hash, nil
}

// CreateBucket creates a new bucket in the storage.
func (f *FileSystem) CreateBucket(
	bucketName string,
) error {
	if err := library.EnforceNamingConventions(bucketName); err != nil {
		return errors.Stamp(err)
	}

	if err := f.createBucket(bucketName); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

// ListBuckets lists all the buckets in the storage and returns their bucketNames,
// after having removed FileSystemBucketPrefix.
func (f *FileSystem) ListBuckets() ([]string, error) {
	return f.listBuckets()
}

func (f *FileSystem) DeleteBucket(
	bucketName string,
) error {
	if err := library.EnforceNamingConventions(bucketName); err != nil {
		return errors.Stamp(err)
	}

	if err := f.deleteBucket(bucketName); err != nil {
		return errors.Stamp(err)
	}

	return nil
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
	artifact *domain.Artifact,
) (*domain.ArtifactStatus, error) {
	if err := library.EnforceNamingConventions(artifact.Name); err != nil {
		return nil, errors.Stamp(err)
	}
	meta, err := f.createMultipartFiles(bucketName, artifact)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return meta, nil
}

// GetMultipartFile retrieves the multipart file recipient from a given bucket and returns its Artifact.
func (f *FileSystem) GetMultipartFile(
	bucketName string,
) (*domain.Artifact, error) {
	meta, err := f.getMultipartFile(bucketName)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return meta, nil
}

// GetMultipartFileStatus retrieves the ArtifactStatus of a multipart file recipient
// based on bucketName and artifactMeta.
func (f *FileSystem) GetMultipartFileStatus(
	bucketName string,
	artifactMeta *domain.Artifact,
) (*domain.ArtifactStatus, error) {
	status, err := f.getArtifactStatus(bucketName, artifactMeta)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return status, nil
}

// DeleteMultipartFile deletes a multipart file recipient from a bucket based on bucketName and artifactMeta.
func (f *FileSystem) DeleteMultipartFile(
	bucketName string,
	artifactMeta *domain.Artifact,
) error {
	if err := f.deleteMultipartFile(bucketName, artifactMeta); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

// WritePartToMultipartFile writes the content of the given part into the multipart file recipient
// on the bucket indicated by the given bucketName.
func (f *FileSystem) WritePartToMultipartFile(bucketName string,
	part *domain.Part,
) (*domain.ArtifactStatus, error) {
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
	artifactMeta *domain.Artifact,
	perm os.FileMode,
) error {
	if err := f.consolidateMultipartFile(bucketName, artifactMeta, perm); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

// GetHashFromFileInfos retrieves the hash of a file from the storage backend.
func (f *FileSystem) GetHashFromFileInfos(filename string) (string, error) {
	if _, ok := f.watchedFileInfos[filename]; !ok {
		return "", errors.From(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			WithDetail("File not found.").
			WithProperty("file_name", filename).
			Throw()
	}

	return f.watchedFileInfos[filename].Hash, nil
}

// Bucket handling methods

func (f *FileSystem) genBucketPath(
	bucketName string,
) string {
	return filepath.Join(f.artifactLocation, library.FileSystemBucketPrefix+bucketName)
}

func (f *FileSystem) createBucket(
	bucketName string,
) error {
	bucketPath := f.genBucketPath(bucketName)
	if err := library.CheckDir(bucketPath); err != nil &&
		!errors.Is(err,
			errors.Intercept(domain.ErrStorageProviderNotFound).
				WithIdentifier(404000).
				Throw()) {
		return errors.Stamp(err)
	}

	if err := os.Mkdir(bucketPath, library.FileSystemDefaultDirMode); err != nil && !os.IsExist(err) {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to create the bucket.").
			WithProperty("bucket_path", bucketPath).
			CausedBy(err).
			Throw()
	}

	return nil
}

func (f *FileSystem) listBuckets() ([]string, error) {
	buckets, err := library.ListDirContentNames(f.artifactLocation, library.BucketFilter)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	if len(buckets) == 0 {
		return nil, errors.From(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			WithDetail("No buckets found.").
			Throw()
	}

	for i, bucket := range buckets {
		buckets[i] = bucket[len(library.FileSystemBucketPrefix):]
	}

	return buckets, nil
}

func (f *FileSystem) deleteBucket(
	bucketName string,
) error {
	bucketPath := f.genBucketPath(bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return errors.Stamp(err)
	}

	if err := os.RemoveAll(bucketPath); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to delete the bucket.").
			WithProperty("bucket_path", bucketPath).
			CausedBy(err).
			Throw()
	}

	return nil
}

func (f *FileSystem) listFiles() ([]string, error) {
	files, err := library.ListDirContentNames(f.artifactLocation, f.interestContentFilter)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return files, nil
}

func (f *FileSystem) deleteFile(
	fileName string,
) error {
	filePath := filepath.Join(f.artifactLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return errors.Stamp(err)
	}

	if err := library.DeleteFile(filePath); err != nil {
		return errors.Stamp(err)
	}

	return nil
}

func (f *FileSystem) hashFile(
	fileName string,
) (string, error) {
	if f.isFileInfoUpToDate(fileName) {
		return f.watchedFileInfos[fileName].Hash, nil
	}

	filePath := filepath.Join(f.artifactLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return "", errors.Stamp(err)
	}

	hash, err := library.HashFile(filePath)
	if err != nil {
		return "", errors.Stamp(err)
	}

	return hash, nil
}

func (f *FileSystem) isFileInfoUpToDate(filename string) bool {
	if f.watchedFileInfos == nil {
		return false
	}

	storedFileInfo, ok := f.watchedFileInfos[filename]
	if !ok {
		return false
	}

	filePath := filepath.Join(f.artifactLocation, filename)

	physicalFileInfo, err := os.Stat(filePath)
	if err != nil {
		f.logger.Error().Err(err).Msg("Failed to get file info.")

		return false
	}

	if storedFileInfo.Size != physicalFileInfo.Size() {
		return false
	}

	if storedFileInfo.LastChangedAt != physicalFileInfo.ModTime() {
		return false
	}

	return true
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

	newFilePath := filepath.Join(f.artifactLocation, newFileName)
	if err := os.Remove(newFilePath); err != nil && !os.IsNotExist(err) {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unexpected error while moving the file to the root location.").
			WithProperty("current_file_path", filePath).
			WithProperty("new_file_path", newFilePath).
			WithProperty("while", "removing existing file from the root location").
			CausedBy(err).
			Throw()
	}

	if err := os.Rename(filePath, newFilePath); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unexpected error while moving the file to the root location.").
			WithProperty("current_file_path", filePath).
			WithProperty("new_file_path", newFilePath).
			WithProperty("while", "moving the file from bucket to root location").
			CausedBy(err).
			Throw()
	}

	return nil
}

// SPEC: The multipart file recipient is composed by three files: the
// artifact metadata, the part index and the recipient

// SPEC: The metadata file has the same name as the final artifact but
// suffixed with ".meta"

// SPEC: The metadata file stores the Artifact structure encoded as JSON

// SPEC: The part index has the same name as the final artifact but suffixed
// with ".parts"

// SPEC: The part index must be created with an empty content

// SPEC: The part index stores the PartMeta structures as binary
// (pkg.go.dev/encoding/binary)

// SPEC: The recipient has the same name as the final artifact but suffixed
// with ".recipient"

// filterOrphansMeta filters the orphaned ArtifactMeta instances from the given
// artifactMetas list and returns the filtered list.
func (f *FileSystem) filterOrphansMeta(
	bucketPath string,
	artifacts []*domain.Artifact,
) []*domain.Artifact {
	filteredArtifacts := make([]*domain.Artifact, 0, len(artifacts))

	for _, artifact := range artifacts {
		if err := library.CheckFile(
			filepath.Join(
				bucketPath,
				artifact.Name+library.FileSystemMultipartPartsSuffix,
			),
		); err != nil {
			f.logger.Warn().Err(err).Msg("The parts file is missing for the multipart file.")

			continue
		}

		if err := library.CheckFile(
			filepath.Join(
				bucketPath,
				artifact.Name+library.FileSystemMultipartRecipientSuffix,
			),
		); err != nil {
			f.logger.Warn().Err(err).Msg("The recipient file is missing for the multipart file.")

			continue
		}

		filteredArtifacts = append(filteredArtifacts, artifact)
	}

	return filteredArtifacts
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

// loadArtifactMeta loads the ArtifactMeta instance from the multipart meta file
// with the given file path.
func loadArtifactMeta(meta *domain.Artifact, filePath string) error {
	metaFile, err := library.GetFile(filePath)
	if err != nil {
		return errors.Stamp(err)
	}

	defer metaFile.Close() // nolint: errcheck // No error check on defer.

	if err := json.NewDecoder(metaFile).Decode(meta); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to load the metadata of the multipart file.").
			WithProperty("file_path", filePath).
			WithProperty("while", "decoding the metadata from json").
			CausedBy(err).
			Throw()
	}

	return nil
}

// getArtifactMetaList returns the list of ArtifactMeta instances in the bucket
// with the given bucketPath.
func (f *FileSystem) getArtifactMeta(bucketPath string) (*domain.Artifact, error) {
	metaFileNames, err := library.ListDirContentNames(bucketPath, library.MultipartMetaFilter)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	artifactMetas := make([]*domain.Artifact, 0, len(metaFileNames))

	for _, metaFileName := range metaFileNames {
		var artifactMeta domain.Artifact

		if err := loadArtifactMeta(&artifactMeta, filepath.Join(bucketPath, metaFileName)); err != nil {
			return nil, errors.Stamp(err)
		}

		artifactMetas = append(artifactMetas, &artifactMeta)
	}

	artifactMetas = f.filterOrphansMeta(bucketPath, artifactMetas)
	if len(artifactMetas) < 1 {
		return nil, errors.From(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			WithDetail("No multipart files found in the bucket.").
			WithProperty("bucket_name", bucketPath).
			Throw()
	}
	if len(artifactMetas) > 1 {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Too many multipart files found in the bucket.").
			WithProperty("bucket_name", bucketPath).
			Throw()
	}

	return artifactMetas[0], nil
}

func (f *FileSystem) createMultipartFiles(
	bucketName string,
	artifactMeta *domain.Artifact,
) (*domain.ArtifactStatus, error) {
	if err := library.CheckDir(f.genBucketPath(bucketName)); err != nil {
		return nil, errors.Stamp(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		artifactMeta.Name,
	)

	err := library.CheckFile(metaFilePath)
	if err == nil {
		artifactStatus, err := f.getArtifactStatus(bucketName, artifactMeta)
		if err != nil {
			return nil, errors.Stamp(err)
		}

		return artifactStatus, nil
	}

	if !errors.Is(err,
		errors.Intercept(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			Throw()) {
		return nil, errors.Stamp(err)
	}

	metaContentBytes, err := json.Marshal(artifactMeta)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to save the artifact metadata.").
			WithProperty("artifact", artifactMeta.Name).
			WithProperty("version", artifactMeta.Version).
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
		artifactMeta.Size,
		library.FileSystemDefaultFileMode,
	); err != nil {
		cleanUp()

		return nil, errors.Stamp(err)
	}

	return &domain.ArtifactStatus{
		Artifact: artifactMeta,
		Parts:    make(map[int64]*domain.PartMeta),
	}, nil
}

func (f *FileSystem) getMultipartFile(bucketName string) (*domain.Artifact, error) {
	bucketPath := f.genBucketPath(bucketName)
	if err := library.CheckDir(bucketPath); err != nil {
		return nil, errors.Stamp(err)
	}

	artifactMeta, err := f.getArtifactMeta(bucketPath)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return artifactMeta, nil
}

func (f *FileSystem) getArtifactStatus(
	bucketName string,
	artifactMeta *domain.Artifact,
) (*domain.ArtifactStatus, error) {
	if err := library.CheckDir(f.genBucketPath(bucketName)); err != nil {
		return nil, errors.Stamp(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		artifactMeta.Name,
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

	var storedArtifactMeta domain.Artifact
	if err := json.NewDecoder(metaFile).Decode(&storedArtifactMeta); err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to load the metadata of the multipart file.").
			WithProperty("file_path", metaFilePath).
			WithProperty("while", "decoding the metadata from json").
			CausedBy(err).
			Throw()
	}

	if err := library.CompareArtifactMetas(artifactMeta, &storedArtifactMeta); err != nil {
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
				WithDetail("Unable to load the parts metadata of the multipart file.").
				WithProperty("file_path", partsFilePath).
				WithProperty("while", "decoding the parts metadata from binary").
				CausedBy(err).
				Throw()
		}

		partMetas[partMeta.Start] = &partMeta
	}

	return &domain.ArtifactStatus{
		Artifact: artifactMeta,
		Parts:    partMetas,
	}, nil
}

func (f *FileSystem) deleteMultipartFile(
	bucketName string,
	artifactMeta *domain.Artifact,
) error {
	if err := library.CheckDir(f.genBucketPath(bucketName)); err != nil {
		return errors.Stamp(err)
	}

	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		artifactMeta.Name,
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
			WithDetail("Unable to delete the multipart file.").
			WithProperties(problems).
			Throw()
	}

	return nil
}

func (f *FileSystem) writePartToMultipartFile(
	bucketName string,
	part *domain.Part,
) (*domain.ArtifactStatus, error) {
	artifactStatus, err := f.getArtifactStatus(bucketName, part.Artifact)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	partsFilePath := f.genMultipartPartsFilePath(bucketName, part.Artifact.Name)
	recipientFilePath := f.genMultipartRecipientFilePath(bucketName, part.Artifact.Name)

	partsFile, err := os.OpenFile(partsFilePath, os.O_WRONLY, library.FileSystemDefaultFileMode)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to open the parts file.").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	defer partsFile.Close() // nolint: errcheck // No error check on defer.

	recipientFile, err := os.OpenFile(recipientFilePath, os.O_RDWR, library.FileSystemDefaultFileMode)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to open the recipient file.").
			WithProperty("file_path", recipientFilePath).
			CausedBy(err).
			Throw()
	}

	defer recipientFile.Close() // nolint: errcheck // No error check on defer.

	_, err = recipientFile.Seek(part.Meta.Start, io.SeekStart)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to move to the start of the part in the recipient file.").
			WithProperty("file_path", recipientFilePath).
			WithProperty("part_start", part.Meta.Start).
			CausedBy(err).
			Throw()
	}

	written, err := io.Copy(recipientFile, part.Content)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to write the part to the recipient file.").
			WithProperty("file_path", recipientFilePath).
			WithProperty("part_size", part.Meta.Size()).
			CausedBy(err).
			Throw()
	}

	if written != part.Meta.Size() {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("The part was not fully written to the recipient file.").
			WithProperty("file_path", recipientFilePath).
			WithProperty("part_size", part.Meta.Size()).
			WithProperty("written", written).
			Throw()
	}

	_, err = partsFile.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to move to the end of the parts file.").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	err = binary.Write(partsFile, binary.LittleEndian, part.Meta)
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to write the part metadata to the parts file.").
			WithProperty("file_path", partsFilePath).
			CausedBy(err).
			Throw()
	}

	artifactStatus.Parts[part.Meta.Start] = part.Meta

	return artifactStatus, nil
}

func (f *FileSystem) consolidateMultipartFile(
	bucketName string,
	artifactMeta *domain.Artifact,
	perm os.FileMode,
) error {
	artifactStatus, err := f.getArtifactStatus(bucketName, artifactMeta)
	if err != nil {
		return err
	}

	if !artifactStatus.IsComplete() {
		return errors.From(domain.ErrStorageProviderBusinessRuleViolation).
			WithIdentifier(422001).
			WithDetail("Unable to consolidate multipart file because it is not complete.").
			WithProperty("bucket_name", bucketName).
			WithProperty("artifact_name", artifactMeta.Name).
			Throw()
	}

	baseFilePath := f.genBaseMultipartFilePath(bucketName, artifactMeta.Name)
	metaFilePath, partsFilePath, recipientFilePath := f.genMultipartFilePaths(
		bucketName,
		artifactMeta.Name,
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
			WithDetail("Unable to calculate the hash of the recipient file.").
			WithProperty("file_path", recipientFilePath).
			CausedBy(err).
			Throw()
	}

	calculedHash := hex.EncodeToString(hasher.Sum(nil))

	if calculedHash != artifactMeta.Hash {
		return errors.From(domain.ErrStorageProviderBusinessRuleViolation).
			WithIdentifier(422001).
			WithDetail("The hash of the recipient file does not match the artifact metadata.").
			WithProperty("component", artifactMeta.Name).
			WithProperty("version", artifactMeta.Version).
			WithProperty("expected_hash", artifactMeta.Hash).
			WithProperty("calculed_hash", calculedHash).
			Throw()
	}

	if err := os.Rename(recipientFilePath, baseFilePath); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to rename the recipient file.").
			WithProperty("from", recipientFilePath).
			WithProperty("to", baseFilePath).
			CausedBy(err).
			Throw()
	}

	if err := os.Chmod(baseFilePath, perm); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			WithDetail("Unable to change the permissions of the recipient file.").
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
			WithDetail("Unable to clean up multipart file bundle.").
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
	return filepath.Join(f.artifactLocation, controlDir)
}

func (f *FileSystem) genWatchedFilesPath() string {
	return filepath.Join(f.genControlDirPath(), watchedFilesInfoName)
}

func (f *FileSystem) genWatchedFileInfo(fileEntry os.DirEntry) (*watchedFileInfo, error) {
	filePath := filepath.Join(f.artifactLocation, fileEntry.Name())

	hash, err := library.HashFile(filePath)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	fileInfo, err := fileEntry.Info()
	if err != nil {
		return nil, errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("Failed to get file info.").
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
				f.logger.Error().Err(err).Msg("Failed to generate watched file info.")

				return
			}

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
			WithDetail("Failed to decode watched file infos.").
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
			WithDetail("Failed to marshal watched file infos.").
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
	fileEntries, err := library.ListDirContent(f.artifactLocation, f.interestContentFilter)
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

func (f *FileSystem) watchFiles(filenameCh chan string) {
	defer f.Done()

	for {
		select {
		case e, ok := <-f.watcher.Events:
			if !ok {
				f.logger.Warn().Msg("Watcher events channel closed.")

				return
			}

			if err := f.updateWatchedFileInfos(f.saveWatchedFileInfosConcurrentSafe); err != nil {
				f.logger.Error().Err(err).Msg("Failed to update watched file infos.")
			}

			// Create an event to trigger a reconcile
			filenameCh <- filepath.Base(e.Name)

		case err, ok := <-f.watcher.Errors:
			if !ok {
				f.logger.Warn().Msg("Watcher errors channel closed.")

				return
			}

			f.logger.Error().Err(err).Msg("Watcher error.")
		}
	}
}

func (f *FileSystem) startWatchFiles(filenameCh chan string) error {
	if err := f.updateWatchedFileInfos(f.saveWatchedFileInfos); err != nil {
		return errors.Stamp(err)
	}

	f.Add(1)

	go f.watchFiles(filenameCh)

	if err := f.watcher.Add(f.artifactLocation); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("Failed to add artifact location to watcher.").
			WithProperty("artifact_location", f.artifactLocation).
			Throw()
	}

	return nil
}
func (f *FileSystem) stopWatchFiles() error {
	defer f.Wait()

	if err := f.watcher.Close(); err != nil {
		return errors.From(domain.ErrStorageProviderInternal).
			WithIdentifier(500000).
			CausedBy(err).
			WithDetail("Failed to close watcher.").
			Throw()
	}

	return nil
}
