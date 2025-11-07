package controller

import (
	"context"
	"time"

	"github.com/rs/zerolog"
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
			gc.logger.Info().Msg("Stopping garbage collector")
			return
		case <-gc.ticker.C:
			gc.logger.Info().Msg("Running garbage collection")
			gc.cleanUnused()
		}
	}
}
func (gc *garbageCollector) Stop() {
	close(gc.done)
}

func (gc *garbageCollector) cleanUnused() {}
