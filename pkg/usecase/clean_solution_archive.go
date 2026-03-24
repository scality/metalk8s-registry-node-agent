package usecase

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type CleanSolutionArchive struct {
	ctx        context.Context
	store      service.StorageProvider
	logger     *zerolog.Logger
	deleteChan chan domain.FileEventDetails
}

func NewCleanSolutionArchive(
	ctx context.Context,
	logger *zerolog.Logger,
	store service.StorageProvider,
	deleteChan chan domain.FileEventDetails,
) *CleanSolutionArchive {
	l := logger.With().Str("use_case", "clean_solution_archive").Logger()
	return &CleanSolutionArchive{
		ctx:        ctx,
		logger:     &l,
		store:      store,
		deleteChan: deleteChan,
	}
}

func (uc *CleanSolutionArchive) Execute() error {
	for eventDetails := range uc.deleteChan {
		switch eventDetails.Origin {
		case domain.SolutionArchivesOrigin:
			uc.logger.Debug().
				Str("path", eventDetails.FullPathName).
				Str("origin", "solution_archives").
				Msg("cleaning unused solution archive")

			err := uc.cleanUnusedSolutionArchives(eventDetails.FullPathName, eventDetails.IsDir)
			if err != nil {
				uc.logger.Error().
					Err(err).
					Str("path", eventDetails.FullPathName).
					Str("origin", "solution_archives").
					Msg("failed to clean unused solution archive")
				uc.store.AddWatchFileOrDirectory(eventDetails.FullPathName) // nolint: errcheck // was existing before
			}
		case domain.SolutionsOrigin:
			err := uc.cleanUnusedSolutions(eventDetails.FullPathName, eventDetails.IsDir)
			if err != nil {
				uc.logger.Error().
					Err(err).
					Str("path", eventDetails.FullPathName).
					Str("origin", "solutions").
					Msg("failed to clean unused solution")
				uc.store.AddWatchFileOrDirectory(eventDetails.FullPathName) // nolint: errcheck // was existing before
			}
		}
	}

	return nil
}

func (uc *CleanSolutionArchive) cleanUnusedSolutions(path string, isDir bool) error {
	uc.store.Lock()
	defer uc.store.Unlock()

	if isDir {
		// Unmount the solution
		err := uc.store.UnmountFile(path)
		if err != nil {
			uc.logger.Error().
				Err(err).
				Str("mount_point", path).
				Msg("failed to unmount unused solution")
			// Continue to try to delete the directory anyway
		}

		// Delete the directory
		err = os.RemoveAll(path)
		if err != nil {
			return errors.From(domain.ErrSolutionArchiveCleanerInternal).
				CausedBy(err).
				WithDetail("failed to delete unused solution directory").
				WithProperty("path", path).
				Throw()
		}

		uc.logger.Debug().
			Str("path", path).
			Msg("finished to clean unused solution directory")

		return nil
	}

	err := library.DeleteFile(path)
	if err != nil {
		return errors.From(domain.ErrSolutionArchiveCleanerInternal).
			CausedBy(err).
			WithDetail("failed to delete unused solution").
			WithProperty("path", path).
			Throw()
	}

	uc.logger.Debug().
		Str("path", path).
		Msg("finished to clean unused solution")

	return nil
}

// FIXME Not sure why this is here, should be in library
func (uc *CleanSolutionArchive) cleanUnusedSolutionArchives(path string, isDir bool) error {
	uc.store.Lock()
	defer uc.store.Unlock()

	if isDir {
		err := os.RemoveAll(path)
		if err != nil {
			return errors.From(domain.ErrSolutionArchiveCleanerInternal).
				CausedBy(err).
				WithDetail("failed to delete unused solution archive directory").
				WithProperty("path", path).
				Throw()
		}

		uc.logger.Debug().
			Str("path", path).
			Msg("finished to clean unused solution archive directory")

		return nil
	}

	err := library.DeleteFile(path)
	if err != nil {
		return errors.From(domain.ErrSolutionArchiveCleanerInternal).
			CausedBy(err).
			WithDetail("failed to delete unused solution archive").
			WithProperty("path", path).
			Throw()
	}

	uc.logger.Debug().
		Str("path", path).
		Msg("finished to clean unused solution archive")

	return nil
}
