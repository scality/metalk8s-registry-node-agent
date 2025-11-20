package sessionremover

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
	l := logger.With().Str("infrastructure", "sessionremover").Logger()
	return &Storage{
		store:  store,
		logger: &l,
	}
}

func (s *Storage) RemoveSession(solutionArchive *domain.SolutionArchive) error {
	s.store.Lock()
	defer s.store.Unlock()

	// List all the buckets
	buckets, err := s.store.ListBuckets()
	if err != nil {
		if errors.Is(err,
			errors.Intercept(domain.ErrSessionRemoverNotFound).
				WithIdentifier(404000).
				Throw()) {
			return nil
		}
		return errors.Stamp(err)
	}

	// Extract the session bucket
	sessionBucket, err := library.ExtractSessionBucket(buckets, solutionArchive)
	if err != nil {
		if errors.Is(err,
			errors.Intercept(domain.ErrSessionRemoverNotFound).
				WithIdentifier(404000).
				Throw()) {
			return nil
		}
		return errors.Intercept(err).
			WithDetail("failed to extract the session bucket").
			Throw()
	}

	err = s.store.DeleteBucket(sessionBucket)
	if err != nil {
		return errors.Stamp(err)
	}

	return nil
}
