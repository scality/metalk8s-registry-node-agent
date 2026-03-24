package usecase

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type UnmountSolutionArchive struct {
	logger      *zerolog.Logger
	locker      sync.Locker
	fileMounter service.FileMounter
}

func NewUnmountSolutionArchive(
	logger *zerolog.Logger,
	locker sync.Locker,
	fileMounter service.FileMounter,
) *UnmountSolutionArchive {
	l := logger.With().Str("use_case", "unmount_solution_archive").Logger()

	return &UnmountSolutionArchive{
		logger:      &l,
		locker:      locker,
		fileMounter: fileMounter,
	}
}

func (uc *UnmountSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Unmounting solution archive")

	uc.locker.Lock()
	defer uc.locker.Unlock()

	err := uc.fileMounter.UnmountFile(library.GenSolutionDirName(solutionArchive))
	if err != nil {
		return errors.Stamp(err)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive unmounting ended")

	return nil
}
