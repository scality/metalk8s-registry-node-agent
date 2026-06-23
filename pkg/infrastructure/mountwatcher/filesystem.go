package mountwatcher

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/moby/sys/mountinfo"
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
	"golang.org/x/sys/unix"
)

const (
	mountInfoPath = "/proc/self/mountinfo"
	// pollTimeoutMillis bounds poll() so the goroutine can observe shutdown and
	// acts as a periodic safety net if a POLLPRI is ever missed.
	pollTimeoutMillis = 600000 // 10 minutes
	// EventTypeUnmount does not match any fsnotify op, so it falls through to
	// handleSolutionDefault (reconcile) in the file events pipeline.
	EventTypeUnmount = "unmount"
	// errorBackoff avoids a hot loop when poll() returns a persistent error.
	errorBackoff = time.Second
)

// FileSystem watches the OS mount table and emits an event whenever a solution
// archive mount point disappears out-of-band (e.g. a manual umount), so the
// controller can remount it. It relies on poll(2) on /proc/self/mountinfo,
// which the kernel marks readable (POLLPRI) on every mount table change.
type FileSystem struct {
	sync.WaitGroup
	knownMu           sync.Mutex
	logger            *zerolog.Logger
	solutionsLocation string
	done              chan struct{}
	// wakeFD is an eventfd used to interrupt a blocked poll() on shutdown so the
	// watch loop observes f.done immediately instead of after pollTimeoutMillis.
	// It is -1 until StartWatchMounts creates it.
	wakeFD int
	// listMounts returns the current OS mount table. It is a field so it can be
	// stubbed in tests; in production it reads /proc/self/mountinfo.
	listMounts func() ([]*mountinfo.Info, error)
	// known maps a currently-mounted solution mount point to its "<name>/<version>".
	known map[string]string
}

var _ service.MountWatcher = &FileSystem{}

func NewFileSystem(logger *zerolog.Logger, solutionsLocation string) service.MountWatcher {
	l := logger.With().
		Str("infrastructure", "mount_watcher").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:            &l,
		solutionsLocation: solutionsLocation,
		done:              make(chan struct{}),
		wakeFD:            -1,
		listMounts:        func() ([]*mountinfo.Info, error) { return mountinfo.GetMounts(nil) },
	}
}

// StartWatchMounts seeds the initial mount snapshot and starts the watch loop.
func (f *FileSystem) StartWatchMounts(filenameChan chan domain.FileEventDetails) error {
	current, err := f.currentSolutionMounts()
	if err != nil {
		return errors.Wrap(domain.ErrMountWatcherInit,
			errors.WithIdentifier(250),
			errors.WithDetail("failed to take initial mount snapshot"),
			errors.CausedBy(err),
		)
	}
	f.knownMu.Lock()
	f.known = current
	f.knownMu.Unlock()

	file, err := os.Open(mountInfoPath)
	if err != nil {
		return errors.Wrap(domain.ErrMountWatcherInit,
			errors.WithIdentifier(252),
			errors.WithDetail("failed to open mountinfo"),
			errors.CausedBy(err),
		)
	}

	// eventfd lets StopWatchMounts wake a blocked poll() immediately.
	wakeFD, err := unix.Eventfd(0, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC)
	if err != nil {
		file.Close() //nolint:errcheck // best effort
		return errors.Wrap(domain.ErrMountWatcherInit,
			errors.WithIdentifier(253),
			errors.WithDetail("failed to create eventfd"),
			errors.CausedBy(err),
		)
	}
	f.wakeFD = wakeFD

	f.Add(1)
	go func() {
		defer file.Close()       //nolint:errcheck // best effort on shutdown
		defer unix.Close(wakeFD) //nolint:errcheck // best effort on shutdown
		f.watchMounts(filenameChan, file)
	}()

	return nil
}

// StopWatchMounts signals the watch loop to stop and waits for it to exit.
func (f *FileSystem) StopWatchMounts() error {
	close(f.done)

	// Wake the blocked poll() so the loop observes f.done immediately rather
	// than after the pollTimeoutMillis ceiling.
	if f.wakeFD >= 0 {
		var buf [8]byte
		buf[7] = 1
		_, _ = unix.Write(f.wakeFD, buf[:]) //nolint:errcheck // best effort on shutdown
	}

	f.Wait()

	return nil
}

// RecordMount registers a controller-initiated mount in the watcher's snapshot.
func (f *FileSystem) RecordMount(mountPoint, objectName string) {
	f.knownMu.Lock()
	defer f.knownMu.Unlock()
	if f.known == nil {
		f.known = make(map[string]string)
	}
	f.known[mountPoint] = objectName
}

// RecordUnmount removes a controller-initiated mount from the watcher's snapshot.
func (f *FileSystem) RecordUnmount(mountPoint string) {
	f.knownMu.Lock()
	defer f.knownMu.Unlock()
	delete(f.known, mountPoint)
}

func (f *FileSystem) watchMounts(filenameChan chan domain.FileEventDetails, file *os.File) {
	defer f.Done()

	fds := []unix.PollFd{
		{Fd: int32(file.Fd()), Events: unix.POLLPRI | unix.POLLERR},
		{Fd: int32(f.wakeFD), Events: unix.POLLIN},
	}

	for {
		select {
		case <-f.done:
			return
		default:
		}

		// poll() returns when the mount table changes (POLLPRI), when
		// StopWatchMounts writes to wakeFD (POLLIN), or on timeout.
		if _, err := unix.Poll(fds, pollTimeoutMillis); err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			f.logger.Error().Err(err).Msg("poll on mountinfo failed")
			time.Sleep(errorBackoff)
			continue
		}

		// Woken for shutdown: loop back so the select observes f.done.
		if fds[1].Revents&unix.POLLIN != 0 {
			continue
		}

		// Re-arm: mountinfo is level-triggered, so we must read it to EOF to
		// clear the pending POLLPRI before the next poll().
		if _, err := file.Seek(0, io.SeekStart); err == nil {
			_, _ = io.Copy(io.Discard, file) //nolint:errcheck // only used to re-arm poll
		}

		f.reconcileMountChanges(filenameChan)
	}
}

// reconcileMountChanges diffs current solution mounts against the last snapshot
// and emits an event for every mount that disappeared.
func (f *FileSystem) reconcileMountChanges(filenameChan chan domain.FileEventDetails) {
	current, err := f.currentSolutionMounts()
	if err != nil {
		f.logger.Error().Err(err).Msg("failed to list solution mounts")
		return
	}

	// Collect the events to emit while holding the lock, then update the
	// snapshot and release the lock before sending. Sending on the unbuffered
	// filenameChan blocks until FileEvents.Listen receives; holding knownMu
	// across that send would stall concurrent RecordMount/RecordUnmount calls
	// from the reconciler.
	events := f.collectDisappearedMounts(current)

	for i := range events {
		select {
		case <-f.done:
			return
		case filenameChan <- events[i]:
		}
	}
}

// collectDisappearedMounts updates the known snapshot to match current and
// returns an event for every solution mount that disappeared since the last
// snapshot. It holds knownMu only for the duration of the diff.
func (f *FileSystem) collectDisappearedMounts(current map[string]string) []domain.FileEventDetails {
	f.knownMu.Lock()
	defer f.knownMu.Unlock()

	var events []domain.FileEventDetails
	for mountPoint, objectName := range f.known {
		if _, stillMounted := current[mountPoint]; stillMounted {
			continue
		}
		f.logger.Info().
			Str("mount_point", mountPoint).
			Str("object_name", objectName).
			Msg("solution mount disappeared, queueing reconcile")

		events = append(events, domain.FileEventDetails{
			FullPathName: mountPoint,
			ObjectName:   objectName,
			IsDir:        true,
			Origin:       domain.SolutionsOrigin,
			EventType:    EventTypeUnmount,
		})
	}

	for mountPoint, objectName := range current {
		f.known[mountPoint] = objectName
	}
	for mountPoint := range f.known {
		if _, stillMounted := current[mountPoint]; !stillMounted {
			delete(f.known, mountPoint)
		}
	}

	return events
}

// currentSolutionMounts returns versioned ("<name>/<version>") mount points
// under the solutions location, keyed by mount point.
func (f *FileSystem) currentSolutionMounts() (map[string]string, error) {
	mounts, err := f.listMounts()
	if err != nil {
		return nil, errors.Wrap(domain.ErrMountWatcherInternal,
			errors.WithIdentifier(251),
			errors.WithDetail("failed to read the mount table"),
			errors.CausedBy(err),
		)
	}

	prefix := f.solutionsLocation + string(os.PathSeparator)
	result := make(map[string]string, len(mounts))
	for _, m := range mounts {
		if !strings.HasPrefix(m.Mountpoint, prefix) {
			continue
		}
		rel, err := filepath.Rel(f.solutionsLocation, m.Mountpoint)
		if err != nil {
			continue
		}
		// Keep only "<name>/<version>" mount points.
		if strings.Count(rel, string(os.PathSeparator)) == 1 {
			result[m.Mountpoint] = rel
		}
	}

	return result, nil
}
