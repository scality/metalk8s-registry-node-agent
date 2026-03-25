package filewatcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

var _ service.FileWatcher = &FileSystem{}

type FileSystem struct {
	watcher *fsnotify.Watcher
}

func NewFileSystem(watcher *fsnotify.Watcher) *FileSystem {
	return &FileSystem{watcher: watcher}
}

func (f *FileSystem) AddWatchFileOrDirectory(path string) error {
	return f.watcher.Add(path)
}

func (f *FileSystem) RemoveWatchFileOrDirectory(path string) error {
	return f.watcher.Remove(path)
}
