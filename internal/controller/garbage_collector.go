package controller

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	metalk8sv1alpha1 "github.com/scality/metalk8s-registry-node-agent/api/v1alpha1"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GarbageCollector implementation
// The GarbageCollector is used to clean unused solutions archives and solutions
// (not related to any NodeSolutionArchive)
type garbageCollector struct {
	ctx       context.Context
	done      chan bool
	logger    *zerolog.Logger
	client    client.Client
	container containerInterface
	ticker    *time.Ticker
	nodeName  string
}

func NewGarbageCollector(
	ctx context.Context,
	l *zerolog.Logger,
	c client.Client,
	container containerInterface,
	interval time.Duration,
	nodeName string,
) *garbageCollector {
	return &garbageCollector{
		done:      make(chan bool),
		logger:    l,
		client:    c,
		container: container,
		ticker:    time.NewTicker(interval),
		nodeName:  nodeName,
	}
}

func (gc *garbageCollector) Run() {
	for {
		select {
		case <-gc.done:
			gc.logger.Info().Msg("stopping garbage collector")
			return
		case <-gc.ticker.C:
			gc.logger.Info().Msg("running garbage collection")
			gc.cleanUnused()
		}
	}
}
func (gc *garbageCollector) Stop() {
	close(gc.done)
}

func (gc *garbageCollector) cleanUnused() {
	nodeSolutionArchiveList := &metalk8sv1alpha1.NodeSolutionArchiveList{}
	if err := gc.client.List(gc.ctx, nodeSolutionArchiveList, client.MatchingFields{"Spec.NodeName": gc.nodeName}); err != nil {
		gc.logger.Error().Err(err).Msg("failed to list custom resources")
		return
	}
	usedSolutionArchives := []*domain.SolutionArchive{}
	for _, item := range nodeSolutionArchiveList.Items {
		gc.logger.Info().Msgf("garbage collecting solution archive: %s", item.Name)
		usedSolutionArchives = append(usedSolutionArchives, &domain.SolutionArchive{
			Name:    item.Spec.Name,
			Version: item.Spec.Version,
		})
	}
	err := gc.container.GetCleanUnusedSolutionArchivesUseCase().Execute(usedSolutionArchives)
	if err != nil {
		gc.logger.Error().Err(err).Msg("failed to clean unused solution archives")
		return
	}

	err = gc.container.GetCleanUnusedSolutionsUseCase().Execute(usedSolutionArchives)
	if err != nil {
		gc.logger.Error().Err(err).Msg("failed to clean unused solutions")
		return
	}
}
