package domain

import "io"

type (
	// Artifact.
	Artifact struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Size    int64  `json:"size"`
		Hash    string `json:"hash"`
	}

	// SessionStatus.
	SessionStatus struct {
		Name               string          `json:"name"`
		Version            string          `json:"version"`
		IncompleteArtifact *ArtifactStatus `json:"artifacts"`
	}

	// ArtifactStatus.
	ArtifactStatus struct {
		Artifact *Artifact           `json:"artifact"`
		Parts    map[int64]*PartMeta `json:"parts"`
	}

	// PartMeta.
	PartMeta struct {
		Start int64
		End   int64
	}

	// Part.
	Part struct {
		Artifact *Artifact
		Meta     *PartMeta
		Content  io.Reader
	}
)

func (as *ArtifactStatus) ReceivedBytes() int64 {
	var received int64
	for _, part := range as.Parts {
		received += part.Size()
	}

	return received
}

func (as *ArtifactStatus) MissingBytes() int64 {
	return as.Artifact.Size - as.ReceivedBytes()
}

func (as *ArtifactStatus) IsComplete() bool {
	return as.MissingBytes() == 0
}

func (pm *PartMeta) Size() int64 {
	return (pm.End - pm.Start) + 1
}
