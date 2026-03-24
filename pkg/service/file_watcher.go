package service

// FileWatcher manages filesystem watch subscriptions.
type FileWatcher interface {
	AddWatchFileOrDirectory(path string) error
	RemoveWatchFileOrDirectory(path string) error
}
