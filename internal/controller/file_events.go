package controller

import (
	"context"

	"github.com/rs/zerolog"
	metalk8sv1alpha1 "github.com/scality/metalk8s-registry-node-agent/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type FileEvents struct {
	ctx          context.Context
	logger       *zerolog.Logger
	client       client.Client
	filenameChan <-chan string
	eventChan    chan event.GenericEvent
	nodeName     string
}

func NewFileEvents(
	ctx context.Context,
	l *zerolog.Logger,
	c client.Client,
	filenameChan <-chan string,
	eventChan chan event.GenericEvent,
	nodeName string,
) *FileEvents {
	return &FileEvents{
		ctx:          ctx,
		logger:       l,
		client:       c,
		filenameChan: filenameChan,
		eventChan:    eventChan,
		nodeName:     nodeName,
	}
}

// ListenForFileEvents runs in a goroutine, processing filenames from a channel
func (f *FileEvents) Listen() {
	// Loop forever, reading from the channel
	for filename := range f.filenameChan {

		// Find the Custom Resource that matches the parsed data
		naList := &metalk8sv1alpha1.NodeSolutionArchiveList{}
		err := f.client.List(f.ctx, naList, client.MatchingFields{"LocalSolutionArchiveNameVersion": filename[:len(filename)-4]})
		if err != nil {
			f.logger.Error().Err(err).Msg("Failed to list custom resources")
			continue
		}

		var foundCR *metalk8sv1alpha1.NodeSolutionArchive
		if len(naList.Items) > 0 {
			foundCR = &naList.Items[0]
		}
		/*
			Careful, multiple cases can be found:
			a) The CR is being deleted, so the solution archive has been deleted => delete event received
			  => No need for another reconcile

			b) The CR is not being deleted, but the solution archive has been deleted (for example manually) => delete event received
			  => We need to reconcile the CR

			c) No CR found (the solution archive has been copied manually) => create event received
			  => No need for another reconcile
		*/
		if foundCR == nil || !foundCR.DeletionTimestamp.IsZero() {
			continue
		}

		f.logger.Info().Msgf("Found matching Custom Resource: %s, queueing for reconcile", foundCR.Name)
		f.eventChan <- event.GenericEvent{
			Object: foundCR,
		}
	}
}
