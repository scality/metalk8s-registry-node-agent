package storageprovider

import (
	"sync"
	"time"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

type (
	watchedFileInfo struct {
		Size          int64     `json:"size"`
		LastChangedAt time.Time `json:"last_changed_at"`
		Hash          string    `json:"hash"`
	}

	WatchedFileStore struct {
		mu    sync.RWMutex
		infos map[string]*watchedFileInfo
	}
)

func NewWatchedFileStore() *WatchedFileStore {
	return &WatchedFileStore{
		infos: make(map[string]*watchedFileInfo),
	}
}

func (s *WatchedFileStore) GetHashFromFileInfos(filename string) (string, error) {
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

func (s *WatchedFileStore) GetSizeFromFileInfos(filename string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, ok := s.infos[filename]
	if !ok {
		return 0, errors.From(domain.ErrStorageProviderNotFound).
			WithIdentifier(404000).
			WithDetail("file not found").
			WithProperty("file_name", filename).
			Throw()
	}

	return info.Size, nil
}

func (s *WatchedFileStore) Set(infos map[string]*watchedFileInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.infos = infos
}
