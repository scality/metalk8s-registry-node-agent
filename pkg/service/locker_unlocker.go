package service

import "github.com/scality/metalk8s-registry-node-agent/pkg/domain"

// LockerUnlocker locks and unlocks objects based on their Solution Archives.
type LockerUnlocker interface {
	// Lock locks an object based on its Solution Archive.
	Lock(solutionArchive *domain.SolutionArchive)

	// Unlock unlocks an object based on its Solution Archive.
	Unlock(solutionArchive *domain.SolutionArchive)
}
