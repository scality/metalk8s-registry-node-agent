package sessionremover

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

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
	l := logger.With().Str("infrastructure", "sessionremover").Logger()
	return &Storage{
		store:  store,
		logger: &l,
	}
}

func (s *Storage) RemoveSession(artifact *domain.Artifact) error {
	s.store.Lock()
	defer s.store.Unlock()

	// List all the buckets
	buckets, err := s.store.ListBuckets()
	if err != nil {
		if errors.Is(err, domain.ErrSessionRemoverNotFoundError) {
			return nil
		}
		return domain.Stamp(err)
	}

	// Extract the session bucket
	sessionBucket, err := library.ExtractSessionBucket(buckets, artifact)
	if err != nil {
		if errors.Is(err, domain.ErrSessionRemoverNotFoundError) {
			return nil
		}
		return errors.Wrap(err, "failed to extract the session bucket")
	}

	err = s.store.DeleteBucket(sessionBucket)
	if err != nil {
		return domain.Stamp(err)
	}

	return nil
}
