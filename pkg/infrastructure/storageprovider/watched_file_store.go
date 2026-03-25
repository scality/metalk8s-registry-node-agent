package storageprovider

import (
	"os"
	"sync"
	"time"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

type (
	WatchedFileInfo struct {
		Size          int64     `json:"size"`
		LastChangedAt time.Time `json:"last_changed_at"`
		Hash          string    `json:"hash"`
	}

	WatchedFileStore struct {
		mu    sync.RWMutex
		infos map[string]*WatchedFileInfo
	}
)

func NewWatchedFileStore() *WatchedFileStore {
	return &WatchedFileStore{
		infos: make(map[string]*WatchedFileInfo),
	}
}

func (s *WatchedFileStore) GetHash(filename string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, ok := s.infos[filename]
	if !ok {
		return "", errors.From(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			WithDetail("file not found").
			WithProperty("file_name", filename).
			Throw()
	}

	return info.Hash, nil
}

func (s *WatchedFileStore) GetSize(filename string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, ok := s.infos[filename]
	if !ok {
		return 0, errors.From(domain.ErrStorageProviderNotFound).
			WithDetail("file not found").
			WithProperty("file_name", filename).
			Throw()
	}

	return info.Size, nil
}

func (s *WatchedFileStore) Get(filename string) (*WatchedFileInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, ok := s.infos[filename]
	return info, ok
}

func (s *WatchedFileStore) IsUpToDate(filename, filePath string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.infos == nil {
		return false
	}

	storedFileInfo, ok := s.infos[filename]
	if !ok {
		return false
	}

	physicalFileInfo, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	if storedFileInfo.Size != physicalFileInfo.Size() {
		return false
	}

	if storedFileInfo.LastChangedAt != physicalFileInfo.ModTime() {
		return false
	}

	return true
}

func (s *WatchedFileStore) Set(infos map[string]*WatchedFileInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.infos = infos
}

func (s *WatchedFileStore) Snapshot() map[string]*WatchedFileInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp := make(map[string]*WatchedFileInfo, len(s.infos))
	for k, v := range s.infos {
		cp[k] = v
	}

	return cp
}
