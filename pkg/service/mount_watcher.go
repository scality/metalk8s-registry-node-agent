package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

// MountWatcher observes the OS mount table and reports changes affecting the
// solutions mount points (e.g. a solution archive unmounted out-of-band).
type MountWatcher interface {
	// StartWatchMounts starts watching the mount table. Disappeared solution
	// mounts are emitted as FileEventDetails on filenameChan so the existing
	// file-event pipeline triggers a reconcile/remount.
	StartWatchMounts(filenameChan chan domain.FileEventDetails) error

	// StopWatchMounts stops the watcher and waits for the goroutine to exit.
	StopWatchMounts() error

	// RecordMount registers a controller-initiated mount in the watcher's snapshot
	// so an out-of-band unmount is detected even before the next poll cycle.
	RecordMount(mountPoint, objectName string)

	// RecordUnmount removes a controller-initiated mount from the snapshot after
	// unmount, so it is not mistaken for an out-of-band disappearance.
	RecordUnmount(mountPoint string)
}
