package usecase

import (
	"context"
	"log/slog"

	"github.com/scality/go-errors"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type RemoveSession struct {
	logger        *slog.Logger
	bucketManager service.BucketManager
	bucketLocker  service.LockerUnlocker
}

func NewRemoveSession(
	logger *slog.Logger,
	bucketManager service.BucketManager,
	bucketLocker service.LockerUnlocker,
) *RemoveSession {
	return &RemoveSession{
		logger:        logger.With(slog.String("use_case", "remove_session")),
		bucketManager: bucketManager,
		bucketLocker:  bucketLocker,
	}
}

func (uc *RemoveSession) Execute(ctx context.Context, solutionArchive *domain.SolutionArchive) error {
	uc.logger.DebugContext(ctx, "Removing session",
		slog.Any("solution_archive", solutionArchive),
	)

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

	uc.logger.DebugContext(ctx, "Session removed",
		slog.Any("solution_archive", solutionArchive),
	)

	return nil
}
