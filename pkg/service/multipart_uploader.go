package service

import (
	"os"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

// MultipartUploader manages the multipart upload lifecycle: create, write
// parts, consolidate, and move the completed file to the root location.
type MultipartUploader interface {
	// CreateMultipartFiles creates into a bucket:
	// - a metadata file
	// - a multipart file recipient
	// - a parts synthesis file
	CreateMultipartFiles(
		bucketName string,
		solutionArchiveMeta *domain.SolutionArchive,
	) (*domain.SolutionArchiveStatus, error)

	// WritePartToMultipartFile properly writes the content of the given part
	// into the multipart file recipient on the bucket indicated by the given
	// bucketName.
	WritePartToMultipartFile(bucketName string, part *domain.Part) (*domain.SolutionArchiveStatus, error)

	// ConsolidateMultipartFile consolidates all the parts of a multipart file
	// in a single flat file into the same bucket it is located.
	//
	// The resulting fileName is the solutionArchiveMeta.FileName appended with the
	// solutionArchiveMeta.Version. The file extension is properly moved to the end.
	// Since the multipart file is consolidated, the parts and all metadata
	// associated are no more available.
	//
	// Also, it will no more appears in the ListMultipartFiles method, but in
	// the ListFilesInBucket instead. That way its content becomes accessible.
	//
	// When supported by the storage backend, the perm parameter is used to set
	// the file permissions.
	ConsolidateMultipartFile(
		bucketName string,
		solutionArchiveMeta *domain.SolutionArchive,
		perm os.FileMode,
	) error

	// MoveFileToRoot moves a file from a bucket to the root location in the
	// storage.
	MoveFileToRoot(bucketName, fileName, newFileName string) error

	// StorePart stores a part into a bucket.
	StorePart(
		sessionBucket string,
		solutionArchiveFromManifest *domain.SolutionArchive,
		part *domain.Part,
	) (*domain.SolutionArchiveStatus, error)
}
