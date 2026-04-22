// nolint: dupl
package domain

import (
	"testing"
)

func TestSolutionArchiveStatus_ContainsPart_beginning(t *testing.T) {
	sas := &SolutionArchiveStatus{
		SolutionArchive: &SolutionArchive{
			Name:    "metalk8s",
			Version: "1.25.5",
			Size:    4000,
			Hash:    "4f55c3c729aa2982c78719f885df3652e02c60787e94fcaf2a222abf4d2de156",
		},
		Parts: map[int64]*PartMeta{
			1000: {Start: 1000, End: 1999},
			2000: {Start: 2000, End: 2999},
			3000: {Start: 3000, End: 3999},
		},
	}

	tests := []struct {
		name string
		part *PartMeta
		want bool
	}{
		{
			name: "part is fully included in the first part from beginning",
			part: &PartMeta{Start: 1000, End: 1732},
			want: true,
		},
		{
			name: "part is fully included in the last part from end",
			part: &PartMeta{Start: 3241, End: 3999},
			want: true,
		},
		{
			name: "part is fully included in the middle part",
			part: &PartMeta{Start: 1500, End: 1625},
			want: true,
		},
		{
			name: "part is fully included in 2 parts with beginning and ending limit",
			part: &PartMeta{Start: 2000, End: 3999},
			want: true,
		},
		{
			name: "part is partly included in 2 parts",
			part: &PartMeta{Start: 1247, End: 2153},
			want: true,
		},
		{
			name: "part is partly included in 2 parts with beginning limit",
			part: &PartMeta{Start: 1000, End: 2153},
			want: true,
		},
		{
			name: "part is partly included in 2 parts with ending limit",
			part: &PartMeta{Start: 2247, End: 3999},
			want: true,
		},
		{
			name: "part is partly included in 3 parts",
			part: &PartMeta{Start: 1247, End: 3153},
			want: true,
		},
		{
			name: "part is fully included in 3 parts",
			part: &PartMeta{Start: 1000, End: 3999},
			want: true,
		},
		{
			name: "part is not included at all",
			part: &PartMeta{Start: 400, End: 699},
			want: false,
		},
		{
			name: "part is over the limits",
			part: &PartMeta{Start: 4000, End: 4699},
			want: false,
		},
		{
			name: "part is partly included in the first part",
			part: &PartMeta{Start: 400, End: 1249},
			want: false,
		},
		{
			name: "part is partly included in the last part",
			part: &PartMeta{Start: 3241, End: 4999},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sas.ContainsPart(test.part)
			if got != test.want {
				t.Errorf("ContainsPart() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSolutionArchiveStatus_ContainsPart_end(t *testing.T) {
	sas := &SolutionArchiveStatus{
		SolutionArchive: &SolutionArchive{
			Name:    "metalk8s",
			Version: "1.25.5",
			Size:    4000,
			Hash:    "4f55c3c729aa2982c78719f885df3652e02c60787e94fcaf2a222abf4d2de156",
		},
		Parts: map[int64]*PartMeta{
			3000: {Start: 0, End: 999},
			1000: {Start: 1000, End: 1999},
			2000: {Start: 2000, End: 2999},
		},
	}

	tests := []struct {
		name string
		part *PartMeta
		want bool
	}{
		{
			name: "part is fully included in the first part from beginning",
			part: &PartMeta{Start: 0, End: 732},
			want: true,
		},
		{
			name: "part is fully included in the last part from end",
			part: &PartMeta{Start: 2241, End: 2999},
			want: true,
		},
		{
			name: "part is fully included in the middle part",
			part: &PartMeta{Start: 1500, End: 1625},
			want: true,
		},
		{
			name: "part is fully included in 2 parts with beginning and ending limit",
			part: &PartMeta{Start: 1000, End: 2999},
			want: true,
		},
		{
			name: "part is partly included in 2 parts",
			part: &PartMeta{Start: 1247, End: 2153},
			want: true,
		},
		{
			name: "part is partly included in 2 parts with beginning limit",
			part: &PartMeta{Start: 0, End: 1153},
			want: true,
		},
		{
			name: "part is partly included in 2 parts with ending limit",
			part: &PartMeta{Start: 1247, End: 2999},
			want: true,
		},
		{
			name: "part is fully included in 2 parts with beginning and ending limit",
			part: &PartMeta{Start: 1000, End: 2999},
			want: true,
		},
		{
			name: "part is partly included in 3 parts",
			part: &PartMeta{Start: 247, End: 2153},
			want: true,
		},
		{
			name: "part is fully included in 3 parts",
			part: &PartMeta{Start: 0, End: 2999},
			want: true,
		},
		{
			name: "part is not included at all",
			part: &PartMeta{Start: 3400, End: 3699},
			want: false,
		},
		{
			name: "part is partly included in the last part",
			part: &PartMeta{Start: 2400, End: 3249},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sas.ContainsPart(test.part)
			if got != test.want {
				t.Errorf("ContainsPart() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSolutionArchiveStatus_ContainsPart_middle(t *testing.T) {
	sas := &SolutionArchiveStatus{
		SolutionArchive: &SolutionArchive{
			Name:    "metalk8s",
			Version: "1.25.5",
			Size:    4000,
			Hash:    "4f55c3c729aa2982c78719f885df3652e02c60787e94fcaf2a222abf4d2de156",
		},
		Parts: map[int64]*PartMeta{
			3000: {Start: 0, End: 999},
			1000: {Start: 1000, End: 1999},
			2000: {Start: 3000, End: 3999},
		},
	}

	tests := []struct {
		name string
		part *PartMeta
		want bool
	}{
		{
			name: "part is fully included in the first part from beginning",
			part: &PartMeta{Start: 0, End: 732},
			want: true,
		},
		{
			name: "part is fully included in the last part from end",
			part: &PartMeta{Start: 3241, End: 3999},
			want: true,
		},
		{
			name: "part is fully included in the middle part",
			part: &PartMeta{Start: 1500, End: 1625},
			want: true,
		},
		{
			name: "part is fully included in 2 parts with beginning and ending limit",
			part: &PartMeta{Start: 0, End: 1999},
			want: true,
		},
		{
			name: "part is partly included in 2 parts",
			part: &PartMeta{Start: 247, End: 1153},
			want: true,
		},
		{
			name: "part is partly included in 2 parts with beginning limit",
			part: &PartMeta{Start: 0, End: 1153},
			want: true,
		},
		{
			name: "part is partly included in 2 parts with ending limit",
			part: &PartMeta{Start: 247, End: 1999},
			want: true,
		},
		{
			name: "part is not included at all",
			part: &PartMeta{Start: 2400, End: 2699},
			want: false,
		},
		{
			name: "part is partly included in the last part",
			part: &PartMeta{Start: 2400, End: 3249},
			want: false,
		},
		{
			name: "part is partly included in the second part",
			part: &PartMeta{Start: 1400, End: 2249},
			want: false,
		},
		{
			name: "missing part is fully included in the given part",
			part: &PartMeta{Start: 1400, End: 3249},
			want: false,
		},
		{
			name: "missing part is fully included in the full part",
			part: &PartMeta{Start: 0, End: 3999},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sas.ContainsPart(test.part)
			if got != test.want {
				t.Errorf("ContainsPart() = %v, want %v", got, test.want)
			}
		})
	}
}
