package usecase

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type RemoveSession struct {
	logger        *zerolog.Logger
	locker        sync.Locker
	bucketManager service.BucketManager
}

func NewRemoveSession(
	logger *zerolog.Logger,
	locker sync.Locker,
	bucketManager service.BucketManager,
) *RemoveSession {
	l := logger.With().Str("use_case", "remove_session").Logger()

	return &RemoveSession{
		logger:        &l,
		locker:        locker,
		bucketManager: bucketManager,
	}
}

func (uc *RemoveSession) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Removing session")

	uc.locker.Lock()
	defer uc.locker.Unlock()

	// List all the buckets
	buckets, err := uc.bucketManager.ListBuckets()
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

	err = uc.bucketManager.DeleteBucket(sessionBucket)
	if err != nil {
		return errors.Stamp(err)
	}

	uc.logger.Debug().Any("solution_archive", solutionArchive).Msg("Session removed")

	return nil
}
