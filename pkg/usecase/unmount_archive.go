package usecase

import (
	"context"
	"log/slog"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type UnmountSolutionArchive struct {
	logger         *slog.Logger
	archiveMounter service.ArchiveMounter
	archiveLocker  service.LockerUnlocker
}

func NewUnmountSolutionArchive(
	logger *slog.Logger,
	archiveMounter service.ArchiveMounter,
	archiveLocker service.LockerUnlocker,
) *UnmountSolutionArchive {
	return &UnmountSolutionArchive{
		logger:         logger.With(slog.String("use_case", "unmount_solution_archive")),
		archiveMounter: archiveMounter,
		archiveLocker:  archiveLocker,
	}
}

func (uc *UnmountSolutionArchive) Execute(ctx context.Context, solutionArchive *domain.SolutionArchive) error {
	uc.logger.DebugContext(ctx, "Unmounting solution archive",
		slog.Any("solution_archive", solutionArchive),
	)

	uc.archiveLocker.Lock(solutionArchive)
	defer uc.archiveLocker.Unlock(solutionArchive)

	err := uc.archiveMounter.UnmountFile(library.GenSolutionDirName(solutionArchive))
	if err != nil {
		return errors.Stamp(err)
	}

	uc.logger.DebugContext(ctx, "Solution archive unmounting ended",
		slog.Any("solution_archive", solutionArchive),
	)

	return nil
}
