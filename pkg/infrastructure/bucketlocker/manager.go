package bucketlocker

import "sync"

type InMemory struct {
	locks sync.Map
}

func NewInMemory() *InMemory {
	return &InMemory{}
}

func (m *InMemory) Lock(bucketName string) {
	m.getOrCreate(bucketName).Lock()
}

func (m *InMemory) Unlock(bucketName string) {
	m.getOrCreate(bucketName).Unlock()
}

func (m *InMemory) getOrCreate(bucketName string) *sync.Mutex {
	val, _ := m.locks.LoadOrStore(bucketName, &sync.Mutex{})
	return val.(*sync.Mutex)
}
