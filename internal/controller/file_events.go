package controller

import (
	"context"

	"github.com/rs/zerolog"
	metalk8sv1alpha1 "github.com/scality/metalk8s-registry-node-agent/api/v1alpha1"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type KubernetesClientInterface interface {
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

type FileEvents struct {
	ctx          context.Context
	logger       *zerolog.Logger
	client       KubernetesClientInterface
	filenameChan <-chan domain.FileEventDetails
	eventChan    chan event.GenericEvent
}

func NewFileEvents(
	ctx context.Context,
	l *zerolog.Logger,
	c KubernetesClientInterface,
	filenameChan <-chan domain.FileEventDetails,
	eventChan chan event.GenericEvent,
) *FileEvents {
	return &FileEvents{
		ctx:          ctx,
		logger:       l,
		client:       c,
		filenameChan: filenameChan,
		eventChan:    eventChan,
	}
}

// ListenForFileEvents runs in a goroutine, processing filenames from a channel
func (f *FileEvents) Listen() {
	// Loop forever, reading from the channel
	for filename := range f.filenameChan {

		// Find the Custom Resource that matches the parsed data
		naList := &metalk8sv1alpha1.NodeSolutionArchiveList{}
		err := f.client.List(f.ctx, naList, client.MatchingFields{"LocalSolutionArchiveNameVersion": filename.ObjectName[:len(filename.ObjectName)-4]})
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
