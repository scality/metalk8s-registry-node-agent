package lockerunlocker

import (
	"sync"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

// To keep the code readable, we don't deal with entry deletion in map.
// In a situation of 10.000 archive/version combination, it uses
// 2(archive+bucket) * 10.000 * 24 bytes (size of a RWMutex) = 480.000 bytes = 480KB
type InMemory struct {
	locks sync.Map
}

func NewInMemory() *InMemory {
	return &InMemory{}
}

var _ service.LockerUnlocker = &InMemory{}

func (m *InMemory) Lock(solutionArchive *domain.SolutionArchive) {
	solutionArchiveVersionedName := library.GetSolutionArchiveNameVersion(solutionArchive.Name, solutionArchive.Version)
	m.getOrCreate(solutionArchiveVersionedName).Lock()
}

func (m *InMemory) Unlock(solutionArchive *domain.SolutionArchive) {
	solutionArchiveVersionedName := library.GetSolutionArchiveNameVersion(solutionArchive.Name, solutionArchive.Version)
	m.getOrCreate(solutionArchiveVersionedName).Unlock()
}

func (m *InMemory) RLock(solutionArchive *domain.SolutionArchive) {
	solutionArchiveVersionedName := library.GetSolutionArchiveNameVersion(solutionArchive.Name, solutionArchive.Version)
	m.getOrCreate(solutionArchiveVersionedName).RLock()
}

func (m *InMemory) RUnlock(solutionArchive *domain.SolutionArchive) {
	solutionArchiveVersionedName := library.GetSolutionArchiveNameVersion(solutionArchive.Name, solutionArchive.Version)
	m.getOrCreate(solutionArchiveVersionedName).RUnlock()
}

func (m *InMemory) getOrCreate(solutionArchiveVersionedName string) *sync.RWMutex {
	val, _ := m.locks.LoadOrStore(solutionArchiveVersionedName, &sync.RWMutex{})
	return val.(*sync.RWMutex)
}
