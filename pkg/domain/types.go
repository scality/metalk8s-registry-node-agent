package domain

import (
	"fmt"
	"io"
	"sort"
)

type FileOrigin int

const (
	SolutionArchivesOrigin FileOrigin = iota + 1
	SolutionsOrigin
)

type (
	// SolutionArchive.
	SolutionArchive struct {
		Name    string  `json:"name"`
		Version string  `json:"version"`
		Size    int64   `json:"size"`
		Hash    *string `json:"hash,omitempty"`
	}

	SolutionArchiveFile struct {
		File          io.ReadCloser `json:"file"`
		Size          int64         `json:"size"`
		ContentLength int64         `json:"content_length"`
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

	FileEventDetails struct {
		FullPathName string
		ObjectName   string
		IsDir        bool
		Origin       FileOrigin
		EventType    string
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

// ContainsPart reports whether every byte in part's range [Start, End] is included
// into the SolutionArchiveStatus
func (sas *SolutionArchiveStatus) ContainsPart(part *PartMeta) bool {
	if part == nil || sas == nil || sas.Parts == nil {
		return false
	}
	partStart := part.Start
	partEnd := part.End
	if partStart > partEnd {
		return false
	}

	// Collect intervals from stored parts
	type interval struct{ start, end int64 }
	var intervals []interval
	for _, pm := range sas.Parts {
		if pm != nil {
			intervals = append(intervals, interval{pm.Start, pm.End})
		}
	}
	if len(intervals) == 0 {
		return false
	}

	// Sort by start, then merge overlapping or adjacent intervals
	sort.Slice(intervals, func(i, j int) bool { return intervals[i].start < intervals[j].start })
	merged := intervals[:1]
	for i := 1; i < len(intervals); i++ {
		last := &merged[len(merged)-1]
		if intervals[i].start <= last.end+1 {
			if intervals[i].end > last.end {
				last.end = intervals[i].end
			}
		} else {
			merged = append(merged, intervals[i])
		}
	}

	for _, iv := range merged {
		// We have before the part so let's go next
		if iv.end < partStart {
			continue
		}
		if iv.start > partStart {
			return false // gap
		}

		// If we reach here it means iv.start <= partStart so we are in the current
		// part that might be the one we want so let's just check this "current" part
		// is long enough to cover our range
		return iv.end >= partEnd
	}
	return false
}

func (sa *SolutionArchive) GetUploadURL(rootAPIPath string) string {
	return fmt.Sprintf("%s/uploads/%s/%s", rootAPIPath, sa.Name, sa.Version)
}

func (sa *SolutionArchive) GetDownloadURL(rootAPIPath string) string {
	return fmt.Sprintf("%s/downloads/%s/%s", rootAPIPath, sa.Name, sa.Version)
}

func (sa *SolutionArchive) GetErrorProperties(usecase string, instance string) map[string]any {
	properties := map[string]any{
		"usecase":                  usecase,
		"solution_archive_name":    sa.Name,
		"solution_archive_version": sa.Version,
	}

	if instance != "" {
		properties["instance"] = instance
	}

	return properties
}
