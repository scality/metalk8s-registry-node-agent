package usecase

import (
	"github.com/rs/zerolog"
	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type RemoveSession struct {
	logger        *zerolog.Logger
	bucketManager service.BucketManager
	bucketLocker  service.LockerUnlocker
}

func NewRemoveSession(
	logger *zerolog.Logger,
	bucketManager service.BucketManager,
	bucketLocker service.LockerUnlocker,
) *RemoveSession {
	l := logger.With().Str("use_case", "remove_session").Logger()

	return &RemoveSession{
		logger:        &l,
		bucketManager: bucketManager,
		bucketLocker:  bucketLocker,
	}
}

func (uc *RemoveSession) Execute(solutionArchive *domain.SolutionArchive) error {
	uc.logger.Debug().
		Any("solution_archive", solutionArchive).
		Msg("Removing session")

	// To avoid simultaneous removals of the same session
	uc.bucketLocker.Lock(solutionArchive)
	defer uc.bucketLocker.Unlock(solutionArchive)

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
