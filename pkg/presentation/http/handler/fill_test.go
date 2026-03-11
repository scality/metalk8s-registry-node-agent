package handler

import (
	"testing"
)

func TestParseRange(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantStart int64
		wantEnd   int64
		wantTotal int64
		wantErr   bool
	}{
		{
			name:      "valid range",
			header:    "bytes=3-19",
			wantStart: 3,
			wantEnd:   19,
			wantTotal: 0,
			wantErr:   false,
		},
		{
			name:      "invalid range, including total",
			header:    "bytes=0-19/20",
			wantStart: 0,
			wantEnd:   0,
			wantTotal: 0,
			wantErr:   true,
		},
		{
			name:      "invalid range, no equal sign",
			header:    "bytes 0-19",
			wantStart: 0,
			wantEnd:   0,
			wantTotal: 0,
			wantErr:   true,
		},
		{
			name:      "invalid range start > end",
			header:    "bytes=19-0",
			wantStart: 0,
			wantEnd:   0,
			wantTotal: 0,
			wantErr:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, end, total, err := parseRange(test.header)
			if (err != nil) != test.wantErr {
				t.Errorf("parseRange() error = %v, wantErr %v", err, test.wantErr)
			}
			if start != test.wantStart {
				t.Errorf("parseRange() = %v, want %v", start, test.wantStart)
			}
			if end != test.wantEnd {
				t.Errorf("parseRange() = %v, want %v", end, test.wantEnd)
			}
			if total != test.wantTotal {
				t.Errorf("parseRange() = %v, want %v", total, test.wantTotal)
			}
		})
	}
}

func TestParseContentRange(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantStart int64
		wantEnd   int64
		wantTotal int64
		wantErr   bool
	}{
		{
			name:      "valid content range",
			header:    "bytes 1-19/200",
			wantStart: 1,
			wantEnd:   19,
			wantTotal: 200,
			wantErr:   false,
		},
		{
			name:      "invalid content range, including equal sign",
			header:    "bytes=0-19/20",
			wantStart: 0,
			wantEnd:   0,
			wantTotal: 0,
			wantErr:   true,
		},
		{
			name:      "invalid content range, missing total",
			header:    "bytes 0-19",
			wantStart: 0,
			wantEnd:   0,
			wantTotal: 0,
			wantErr:   true,
		},
		{
			name:      "invalid content range, start > end",
			header:    "bytes 19-0/20",
			wantStart: 0,
			wantEnd:   0,
			wantTotal: 0,
			wantErr:   true,
		},
		{
			name:      "invalid content range, total != end-start+1",
			header:    "bytes 0-19/19",
			wantStart: 0,
			wantEnd:   0,
			wantTotal: 0,
			wantErr:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, end, total, err := parseContentRange(test.header)
			if (err != nil) != test.wantErr {
				t.Errorf("parseContentRange() error = %v, wantErr %v", err, test.wantErr)
			}
			if start != test.wantStart {
				t.Errorf("parseContentRange() = %v, want %v", start, test.wantStart)
			}
			if end != test.wantEnd {
				t.Errorf("parseContentRange() = %v, want %v", end, test.wantEnd)
			}
			if total != test.wantTotal {
				t.Errorf("parseContentRange() = %v, want %v", total, test.wantTotal)
			}
		})
	}
}
