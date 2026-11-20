package usecase

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"sync"
	"testing"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

type fakeCleaner struct {
	mu        sync.Mutex
	archives  []string
	solutions []string
}

func (f *fakeCleaner) CleanUnusedSolutionArchives(_ context.Context, path string, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.archives = append(f.archives, path)
	return nil
}

func (f *fakeCleaner) CleanUnusedSolutions(_ context.Context, path string, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.solutions = append(f.solutions, path)
	return nil
}

type fakeFileWatcher struct{}

func (fakeFileWatcher) Init() error                                        { return nil }
func (fakeFileWatcher) AddWatchFileOrDirectory(string) error               { return nil }
func (fakeFileWatcher) RemoveWatchFileOrDirectory(string) error            { return nil }
func (fakeFileWatcher) StartWatchFiles(chan domain.FileEventDetails) error { return nil }
func (fakeFileWatcher) StopWatchFiles() error                              { return nil }

func runCleanArchive(t *testing.T, disabled bool) *fakeCleaner {
	t.Helper()

	cleaner := &fakeCleaner{}
	deleteChan := make(chan domain.FileEventDetails)
	uc := NewCleanArchive(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		cleaner,
		fakeFileWatcher{},
		deleteChan,
		disabled,
	)

	done := make(chan struct{})
	go func() {
		uc.Execute(context.Background())
		close(done)
	}()

	deleteChan <- domain.FileEventDetails{
		Origin:       domain.SolutionArchivesOrigin,
		FullPathName: "/archives/metalk8s-1.25.3.iso",
	}
	deleteChan <- domain.FileEventDetails{
		Origin:       domain.SolutionsOrigin,
		FullPathName: "/solutions/metalk8s/1.25.3",
	}
	close(deleteChan)
	<-done

	return cleaner
}

func TestCleanArchiveEnabled(t *testing.T) {
	cleaner := runCleanArchive(t, false)

	wantArchives := []string{"/archives/metalk8s-1.25.3.iso"}
	if !slices.Equal(cleaner.archives, wantArchives) {
		t.Errorf("archives cleaned: got %q, want %q", cleaner.archives, wantArchives)
	}
	wantSolutions := []string{"/solutions/metalk8s/1.25.3"}
	if !slices.Equal(cleaner.solutions, wantSolutions) {
		t.Errorf("solutions cleaned: got %q, want %q", cleaner.solutions, wantSolutions)
	}
}

func TestCleanArchiveDisabled(t *testing.T) {
	cleaner := runCleanArchive(t, true)

	if len(cleaner.archives) != 0 {
		t.Errorf("archives cleaned: got %q, want none", cleaner.archives)
	}
	if len(cleaner.solutions) != 0 {
		t.Errorf("solutions cleaned: got %q, want none", cleaner.solutions)
	}
}
