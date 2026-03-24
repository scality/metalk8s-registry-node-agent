package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

// MultipartInspector provides read and delete operations on existing
// multipart uploads.
type MultipartInspector interface {
	GetMultipartFile(bucketName string) (*domain.SolutionArchive, error)
	GetMultipartFileStatus(bucketName string, solutionArchiveMeta *domain.SolutionArchive) (*domain.SolutionArchiveStatus, error)
	DeleteMultipartFile(bucketName string, solutionArchiveMeta *domain.SolutionArchive) error
}
