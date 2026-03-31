package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

// MultipartInspector provides read and delete operations on existing
// multipart uploads.
type MultipartInspector interface {
	// GetMultipartFile retrieves the multipart file recipient from a given
	// bucket and returns its SolutionArchiveMeta.
	GetMultipartFile(bucketName string) (*domain.SolutionArchive, error)

	// GetMultipartFileStatus retrieves the SolutionArchiveStatus of a multipart file
	// recipient based on bucketName and solutionArchiveMeta.
	GetMultipartFileStatus(
		bucketName string,
		solutionArchiveMeta *domain.SolutionArchive,
	) (*domain.SolutionArchiveStatus, error)
}
