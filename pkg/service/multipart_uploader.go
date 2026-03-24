package service

import (
	"os"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

// MultipartUploader manages the multipart upload lifecycle: create, write
// parts, consolidate, and move the completed file to the root location.
type MultipartUploader interface {
	CreateMultipartFiles(bucketName string, solutionArchiveMeta *domain.SolutionArchive) (*domain.SolutionArchiveStatus, error)
	WritePartToMultipartFile(bucketName string, part *domain.Part) (*domain.SolutionArchiveStatus, error)
	ConsolidateMultipartFile(bucketName string, solutionArchiveMeta *domain.SolutionArchive, perm os.FileMode) error
	MoveFileToRoot(bucketName, fileName, newFileName string) error
}
