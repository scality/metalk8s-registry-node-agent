package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

// FileWatcher manages filesystem watch subscriptions.
type FileWatcher interface {
	// Init initializes the file watcher.
	Init() error

	// AddWatchFileOrDirectory adds a file or directory to the watcher.
	AddWatchFileOrDirectory(path string) error

	// RemoveWatchFileOrDirectory removes a file or directory from the watcher.
	RemoveWatchFileOrDirectory(path string) error

	// StartWatchFiles starts the watcher on the storage provider.
	StartWatchFiles(filenameChan chan domain.FileEventDetails) error

	// Stop stops the watcher on the storage provider.
	StopWatchFiles() error
}
