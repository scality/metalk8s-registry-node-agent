package domain

// IntegrityStage identifies where an integrity check of a solution archive failed.
type IntegrityStage string

const (
	IntegrityStageChunkDigest     IntegrityStage = "chunk_digest"
	IntegrityStageArchiveChecksum IntegrityStage = "archive_checksum"
)
