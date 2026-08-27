//nolint:goconst
package mountwatcher

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/moby/sys/mountinfo"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

func newTestFS(solutionsLocation string, mounts []*mountinfo.Info) *FileSystem {
	return &FileSystem{
		logger:            slog.New(slog.DiscardHandler),
		solutionsLocation: solutionsLocation,
		done:              make(chan struct{}),
		listMounts:        func() ([]*mountinfo.Info, error) { return mounts, nil },
		known:             map[string]string{},
	}
}

func mountAt(mountPoint string) *mountinfo.Info {
	return &mountinfo.Info{Mountpoint: mountPoint}
}

func TestCurrentSolutionMounts(t *testing.T) {
	f := newTestFS("/solutions", []*mountinfo.Info{
		mountAt("/"),
		mountAt("/solutions"),                       // root, not versioned -> excluded
		mountAt("/solutions/metalk8s"),              // name only -> excluded
		mountAt("/solutions/metalk8s/1.25.3"),       // versioned -> included
		mountAt("/solutions/other/2.0.0"),           // versioned -> included
		mountAt("/solutions/metalk8s/1.25.3/extra"), // too deep -> excluded
		mountAt("/archives/foo"),                    // wrong prefix -> excluded
		mountAt("/solutionsbis/metalk8s/1.0.0"),     // prefix is not a path boundary -> excluded
	})

	got, err := f.currentSolutionMounts()
	if err != nil {
		t.Fatalf("currentSolutionMounts() returned an unexpected error: %v", err)
	}

	want := map[string]string{
		"/solutions/metalk8s/1.25.3": "metalk8s/1.25.3",
		"/solutions/other/2.0.0":     "other/2.0.0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("currentSolutionMounts() = %v, want %v", got, want)
	}
}

func TestReconcileMountChanges(t *testing.T) {
	tests := []struct {
		name       string
		known      map[string]string
		current    []*mountinfo.Info
		wantEvents []domain.FileEventDetails
		wantKnown  map[string]string
	}{
		{
			name: "mount disappeared emits an unmount event",
			known: map[string]string{
				"/solutions/metalk8s/1.25.3": "metalk8s/1.25.3",
			},
			current: nil,
			wantEvents: []domain.FileEventDetails{
				{
					FullPathName: "/solutions/metalk8s/1.25.3",
					ObjectName:   "metalk8s/1.25.3",
					IsDir:        true,
					Origin:       domain.SolutionsOrigin,
					EventType:    EventTypeUnmount,
				},
			},
			wantKnown: map[string]string{},
		},
		{
			name: "mount still present emits nothing",
			known: map[string]string{
				"/solutions/metalk8s/1.25.3": "metalk8s/1.25.3",
			},
			current:    []*mountinfo.Info{mountAt("/solutions/metalk8s/1.25.3")},
			wantEvents: nil,
			wantKnown: map[string]string{
				"/solutions/metalk8s/1.25.3": "metalk8s/1.25.3",
			},
		},
		{
			name:    "new mount appearing emits nothing but updates the snapshot",
			known:   map[string]string{},
			current: []*mountinfo.Info{mountAt("/solutions/metalk8s/1.25.3")},
			wantKnown: map[string]string{
				"/solutions/metalk8s/1.25.3": "metalk8s/1.25.3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTestFS("/solutions", tt.current)
			f.known = tt.known

			ch := make(chan domain.FileEventDetails, 10)
			f.reconcileMountChanges(ch)
			close(ch)

			var gotEvents []domain.FileEventDetails
			for ev := range ch {
				gotEvents = append(gotEvents, ev)
			}

			if !reflect.DeepEqual(gotEvents, tt.wantEvents) {
				t.Errorf("emitted events = %v, want %v", gotEvents, tt.wantEvents)
			}
			if !reflect.DeepEqual(f.known, tt.wantKnown) {
				t.Errorf("known snapshot = %v, want %v", f.known, tt.wantKnown)
			}
		})
	}
}

func TestRecordMount(t *testing.T) {
	f := newTestFS("/solutions", nil)

	f.RecordMount("/solutions/metalk8s/1.25.3", "metalk8s/1.25.3")

	ch := make(chan domain.FileEventDetails, 10)
	f.reconcileMountChanges(ch)
	close(ch)

	var gotEvents []domain.FileEventDetails
	for ev := range ch {
		gotEvents = append(gotEvents, ev)
	}

	wantEvents := []domain.FileEventDetails{
		{
			FullPathName: "/solutions/metalk8s/1.25.3",
			ObjectName:   "metalk8s/1.25.3",
			IsDir:        true,
			Origin:       domain.SolutionsOrigin,
			EventType:    EventTypeUnmount,
		},
	}
	if !reflect.DeepEqual(gotEvents, wantEvents) {
		t.Errorf("emitted events = %v, want %v", gotEvents, wantEvents)
	}
}
