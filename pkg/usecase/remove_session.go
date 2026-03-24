package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type RemoveSession struct {
	logger *zerolog.Logger
	store  service.StorageProvider
}

func NewRemoveSession(
	logger *zerolog.Logger,
	store service.StorageProvider,
) *RemoveSession {
	l := logger.With().Str("use_case", "remove_session").Logger()

	return &RemoveSession{
		logger: &l,
		store:  store,
	}
}

func (uc *RemoveSession) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Removing session")

	uc.store.Lock()
	defer uc.store.Unlock()

	// List all the buckets
	buckets, err := uc.store.ListBuckets()
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

	err = uc.store.DeleteBucket(sessionBucket)
	if err != nil {
		return errors.Stamp(err)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Session removed")

	return nil
}
