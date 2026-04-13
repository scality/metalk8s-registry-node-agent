package service

import (
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

// MultipartRemover manages the multipart removal.
type MultipartRemover interface {
	// DeleteMultipartFile deletes a multipart file recipient from a bucket
	// based in given bucketName and solutionArchiveMeta.
	DeleteMultipartFile(bucketName string, solutionArchiveMeta *domain.SolutionArchive) error
}
