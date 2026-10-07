package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

// MetricsRecorder records domain events for observability.
// Implementations must be safe for concurrent use and must never fail the caller.
type MetricsRecorder interface {
	// SetArchiveState reports the state flags of a solution archive on this node.
	SetArchiveState(solutionArchive *domain.SolutionArchive, initialized, available, served bool)
	// ForgetArchive drops every series related to a solution archive.
	ForgetArchive(solutionArchive *domain.SolutionArchive)
	// IncIntegrityFailure counts a failed integrity check of a solution archive.
	IncIntegrityFailure(solutionArchive *domain.SolutionArchive, stage domain.IntegrityStage)
	// IncUploadRequest counts an upload request targeting a solution archive,
	// answered with the given HTTP status code. Archives never reported through
	// SetArchiveState, or forgotten since, are counted without being identified.
	IncUploadRequest(solutionArchive *domain.SolutionArchive, statusCode int)
}
