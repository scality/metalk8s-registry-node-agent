package solutioncleaner

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type Storage struct {
	store            service.StorageProvider
	logger           *zerolog.Logger
	solutionLocation string
}

func NewStorage(
	store service.StorageProvider,
	logger *zerolog.Logger,
	solutionLocation string,
) *Storage {
	l := logger.With().Str("infrastructure", "solution_cleaner").Logger()
	return &Storage{
		store:            store,
		logger:           &l,
		solutionLocation: solutionLocation,
	}
}

func (s *Storage) CleanUnusedSolutions(usedSolutionArchives []*domain.SolutionArchive) error {
	s.store.Lock()
	defer s.store.Unlock()

	// Create a map of used solution directories for quick lookup
	usedSolutions := make(map[string]struct{})
	for _, solutionArchive := range usedSolutionArchives {
		solutionDir := library.GenSolutionDirName(solutionArchive)
		usedSolutions[solutionDir] = struct{}{}
	}

	// List all name directories in the solution location
	entries, err := os.ReadDir(s.solutionLocation)
	if err != nil {
		if os.IsNotExist(err) {
			// Solution location doesn't exist, nothing to clean
			return nil
		}
		return errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("failed to read solution location").
			WithProperty("solution_location", s.solutionLocation).
			CausedBy(err).
			Throw()
	}

	// Iterate through name directories
	for _, nameEntry := range entries {
		if !nameEntry.IsDir() {
			// Delete unexpected files in solution location
			unexpectedFilePath := filepath.Join(s.solutionLocation, nameEntry.Name())
			s.logger.Debug().
				Str("file_path", unexpectedFilePath).
				Msg("deleting unexpected file in solution location")

			err := os.Remove(unexpectedFilePath)
			if err != nil {
				s.logger.Error().
					Err(err).
					Str("file_path", unexpectedFilePath).
					Msg("failed to delete unexpected file")
			}
			continue
		}

		namePath := filepath.Join(s.solutionLocation, nameEntry.Name())
		versionEntries, err := os.ReadDir(namePath)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("name_path", namePath).
				Msg("failed to read version directories")
			continue
		}

		// Iterate through version directories
		for _, versionEntry := range versionEntries {
			if !versionEntry.IsDir() {
				// Delete unexpected files in name directory
				unexpectedFilePath := filepath.Join(namePath, versionEntry.Name())
				s.logger.Debug().
					Str("file_path", unexpectedFilePath).
					Msg("deleting unexpected file in name directory")

				err := os.Remove(unexpectedFilePath)
				if err != nil {
					s.logger.Error().
						Err(err).
						Str("file_path", unexpectedFilePath).
						Msg("failed to delete unexpected file")
				}
				continue
			}

			solutionDir := filepath.Join(nameEntry.Name(), versionEntry.Name())
			mountPoint := solutionDir

			// Check if this solution is in the used list
			if _, isUsed := usedSolutions[solutionDir]; !isUsed {
				s.logger.Debug().
					Str("solution_dir", solutionDir).
					Msg("unmounting and deleting unused solution")

				// Unmount the solution
				err := s.store.UnmountFile(mountPoint)
				if err != nil {
					s.logger.Error().
						Err(err).
						Str("mount_point", mountPoint).
						Msg("failed to unmount unused solution")
					// Continue to try to delete the directory anyway
				}

				// Delete the directory
				versionPath := filepath.Join(namePath, versionEntry.Name())
				err = os.RemoveAll(versionPath)
				if err != nil {
					s.logger.Error().
						Err(err).
						Str("version_path", versionPath).
						Msg("failed to delete unused solution directory")
					continue
				}

				s.logger.Info().
					Str("solution_dir", solutionDir).
					Msg("deleted unused solution")
			}
		}

		// Clean up empty name directories
		isEmpty, err := library.IsEmpty(namePath)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("name_path", namePath).
				Msg("failed to check if name directory is empty")
			continue
		}

		if isEmpty {
			err = os.Remove(namePath)
			if err != nil {
				s.logger.Error().
					Err(err).
					Str("name_path", namePath).
					Msg("failed to delete empty name directory")
			} else {
				s.logger.Debug().
					Str("name_path", namePath).
					Msg("deleted empty name directory")
			}
		}
	}

	return nil
}
