package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type UnmountSolutionArchive struct {
	logger *zerolog.Logger
	store  service.StorageProvider
}

func NewUnmountSolutionArchive(
	logger *zerolog.Logger,
	store service.StorageProvider,
) *UnmountSolutionArchive {
	l := logger.With().Str("use_case", "unmount_solution_archive").Logger()

	return &UnmountSolutionArchive{
		logger: &l,
		store:  store,
	}
}

func (uc *UnmountSolutionArchive) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Unmounting solution archive")

	uc.store.Lock()
	defer uc.store.Unlock()

	err := uc.store.UnmountFile(library.GenSolutionDirName(solutionArchive))
	if err != nil {
		return errors.Stamp(err)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Solution archive unmounting ended")

	return nil
}
