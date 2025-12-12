package solutionarchivecleaner

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type FileSystem struct {
	ctx        context.Context
	store      service.StorageProvider
	logger     *zerolog.Logger
	deleteChan chan domain.FileEventDetails
}

func NewFileSystem(
	ctx context.Context,
	logger *zerolog.Logger,
	store service.StorageProvider,
	deleteChan chan domain.FileEventDetails,
) *FileSystem {
	l := logger.With().Str("infrastructure", "solution_archive_cleaner").Logger()
	return &FileSystem{
		ctx:        ctx,
		logger:     &l,
		store:      store,
		deleteChan: deleteChan,
	}
}

func (cl *FileSystem) Run() {
	for eventDetails := range cl.deleteChan {
		switch eventDetails.Origin {
		case domain.SolutionArchivesOrigin:
			err := cl.cleanUnusedSolutionArchives(eventDetails.FullPathName, eventDetails.IsDir)
			if err != nil {
				cl.logger.Error().
					Err(err).
					Str("path", eventDetails.FullPathName).
					Str("origin", "solution_archives").
					Msg("failed to clean unused solution archive")
				cl.store.AddWatchFileOrDirectory(eventDetails.FullPathName) // nolint: errcheck // was existing before
			}
		case domain.SolutionsOrigin:
			err := cl.cleanUnusedSolutions(eventDetails.FullPathName, eventDetails.IsDir)
			if err != nil {
				cl.logger.Error().
					Err(err).
					Str("path", eventDetails.FullPathName).
					Str("origin", "solutions").
					Msg("failed to clean unused solution")
				cl.store.AddWatchFileOrDirectory(eventDetails.FullPathName) // nolint: errcheck // was existing before
			}
		}
	}
}

func (cl *FileSystem) cleanUnusedSolutions(path string, isDir bool) error {
	cl.store.Lock()
	defer cl.store.Unlock()

	if isDir {
		// Unmount the solution
		err := cl.store.UnmountFile(path)
		if err != nil {
			cl.logger.Error().
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

		cl.logger.Debug().
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

	cl.logger.Debug().
		Str("path", path).
		Msg("finished to clean unused solution")

	return nil
}

func (cl *FileSystem) cleanUnusedSolutionArchives(path string, isDir bool) error {
	cl.store.Lock()
	defer cl.store.Unlock()

	if isDir {
		err := os.RemoveAll(path)
		if err != nil {
			return errors.From(domain.ErrSolutionArchiveCleanerInternal).
				CausedBy(err).
				WithDetail("failed to delete unused solution archive directory").
				WithProperty("path", path).
				Throw()
		}

		cl.logger.Debug().
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

	cl.logger.Debug().
		Str("path", path).
		Msg("finished to clean unused solution archive")

	return nil
}
