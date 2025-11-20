package artifactvalidator

import (
	"github.com/rs/zerolog"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type Storage struct {
	store  service.StorageProvider
	logger *zerolog.Logger
}

func NewStorage(
	store service.StorageProvider,
	logger *zerolog.Logger,
) *Storage {
	l := logger.With().Str("infrastructure", "artifactvalidator").Logger()
	return &Storage{
		store:  store,
		logger: &l,
	}
}

func (s *Storage) ValidateArtifact(artifact *domain.Artifact) (bool, error) {
	s.store.Lock()
	defer s.store.Unlock()

	// List all artifacts in the storage
	// matching artifactStorageNamePattern
	fileNames, err := s.store.ListFiles()
	if err != nil {
		return false, errors.Stamp(err)
	}

	// Check if the artifact exists in the storage
	if !library.ArtifactExists(artifact, fileNames) {
		return false, nil
	}

	hash, err := s.store.GetHashFromFileInfos(library.GenArtifactFileName(artifact))
	if err != nil {
		return false, errors.Stamp(err)
	}
	if hash != artifact.Hash {
		err := s.store.DeleteFile(library.GenArtifactFileName(artifact))
		if err != nil {
			return false, errors.Stamp(err)
		}
		return false, nil
	}

	return true, nil
}
