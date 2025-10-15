package domain

import "io"

type (
	// SolutionArchive.
	SolutionArchive struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Size    int64  `json:"size"`
		Hash    string `json:"hash"`
	}

	SolutionArchiveFile struct {
		File io.ReadCloser `json:"file"`
		Size int64         `json:"size"`
	}

	// SessionStatus.
	SessionStatus struct {
		Name                      string                 `json:"name"`
		Version                   string                 `json:"version"`
		IncompleteSolutionArchive *SolutionArchiveStatus `json:"solutionArchives"`
	}

	// SolutionArchiveStatus.
	SolutionArchiveStatus struct {
		SolutionArchive *SolutionArchive    `json:"solutionArchive"`
		Parts           map[int64]*PartMeta `json:"parts"`
	}

	// PartMeta.
	PartMeta struct {
		Start int64
		End   int64
	}

	// Part.
	Part struct {
		SolutionArchive *SolutionArchive
		Meta            *PartMeta
		Content         io.Reader
	}
)

func (sas *SolutionArchiveStatus) ReceivedBytes() int64 {
	var received int64
	for _, part := range sas.Parts {
		received += part.Size()
	}

	return received
}

func (sas *SolutionArchiveStatus) MissingBytes() int64 {
	return sas.SolutionArchive.Size - sas.ReceivedBytes()
}

func (sas *SolutionArchiveStatus) IsComplete() bool {
	return sas.MissingBytes() == 0
}

func (pm *PartMeta) Size() int64 {
	return (pm.End - pm.Start) + 1
}
