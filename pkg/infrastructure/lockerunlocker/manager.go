package lockerunlocker

import (
	"sync"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

// To keep the code readable, we don't deal with entry deletion in map.
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

func (m *InMemory) getOrCreate(solutionArchiveVersionedName string) *sync.Mutex {
	val, _ := m.locks.LoadOrStore(solutionArchiveVersionedName, &sync.Mutex{})
	return val.(*sync.Mutex)
}
